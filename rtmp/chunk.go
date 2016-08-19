package rtmp

import (
	"bufio"
	"encoding/binary"
	"io"
	"sync"
)

const (
	fmtFull   = uint8(0x00)
	fmtStream = uint8(0x01)
	fmtDelta  = uint8(0x02)
	fmtData   = uint8(0x03)
)

const timeOverflow = int64(0xffffff)

func getUint24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

var be = binary.BigEndian
var le = binary.LittleEndian

type chunk struct {
	Id     uint32
	Time   int64
	Type   uint8
	Stream uint32
	Data   []byte
}

type reader struct {
	buf  *bufio.Reader
	mux  map[uint32]*chunkReader
	skip int
	size int
}

func newReader(r io.Reader) *reader {
	return &reader{
		buf:  bufio.NewReaderSize(r, bufferSize),
		mux:  make(map[uint32]*chunkReader),
		size: 128,
	}
}

func (r *reader) ReadChunk() (ch *chunk, err error) {
	var fmt uint8
	var id uint32
	for {
		r.discard()
		if fmt, id, err = r.readHeader(); err != nil {
			return nil, err
		}
		cr, ok := r.mux[id]
		if !ok {
			cr = new(chunkReader)
			cr.Id = id
			r.mux[id] = cr
		}
		if ch, err = cr.Read(r, fmt); ch != nil || err != nil {
			return
		}
	}
}

func (r *reader) Peek(n int) ([]byte, error) {
	r.discard()
	r.skip = n
	return r.buf.Peek(n)
}

func (r *reader) Read(b []byte) (int, error) {
	r.discard()
	return r.buf.Read(b)
}

func (r *reader) discard() {
	if r.skip > 0 {
		r.buf.Discard(r.skip)
		r.skip = 0
	}
}

func (r *reader) readHeader() (fmt uint8, id uint32, err error) {
	var b byte
	if b, err = r.buf.ReadByte(); err != nil {
		return
	}
	fmt, id = b>>6&0x03, uint32(b&0x3f)
	if id == 0 {
		if b, err = r.buf.ReadByte(); err != nil {
			return
		}
		id = uint32(b) + 64
	} else if id == 1 {
		a, _ := r.buf.ReadByte()
		if b, err = r.buf.ReadByte(); err != nil {
			return
		}
		id = uint32(a)<<8 + uint32(b) + 64
	}
	return
}

type chunkReader struct {
	chunk
	buf []byte
	pos int
	len int
}

func (cr *chunkReader) Reset(n int) {
	if len(cr.buf) < n {
		cr.buf = make([]byte, (1+(n>>8))<<8)
	}
	cr.len, cr.pos = n, 0
}

func (cr *chunkReader) Read(r *reader, fmt uint8) (ch *chunk, err error) {
	switch fmt {
	case fmtFull:
		err = cr.readFullHeader(r)
	case fmtStream:
		err = cr.readStreamHeader(r)
	case fmtDelta:
		err = cr.readDeltaHeader(r)
	}
	if err != nil {
		return
	}
	return cr.readPayload(r)
}

func (cr *chunkReader) readPayload(r *reader) (ch *chunk, err error) {
	n := cr.len - cr.pos
	if n > r.size {
		n = r.size
	} else if cr.pos == 0 && cr.len <= bufferSize {
		if cr.Data, err = r.Peek(cr.len); err != nil {
			return
		}
		ch = &cr.chunk
		return
	}
	off := cr.pos + n
	if _, err = r.Read(cr.buf[cr.pos:off]); err != nil {
		return
	}
	if off == cr.len {
		cr.pos, cr.Data, ch = 0, cr.buf[:off], &cr.chunk
	} else {
		cr.pos = off
	}
	return
}

func (cr *chunkReader) readFullHeader(r *reader) (err error) {
	var b []byte
	if b, err = r.Peek(11); err != nil {
		return
	}
	cr.Time = int64(getUint24(b))
	cr.Reset(int(getUint24(b[3:])))
	cr.Type = b[6]
	cr.Stream = le.Uint32(b[7:])
	if cr.Time == timeOverflow {
		if b, err = r.Peek(4); err != nil {
			return
		}
		cr.Time = int64(be.Uint32(b))
	}
	return
}

func (cr *chunkReader) readStreamHeader(r *reader) (err error) {
	var b []byte
	if b, err = r.Peek(7); err != nil {
		return
	}
	dt := int64(getUint24(b))
	cr.Reset(int(getUint24(b[3:])))
	cr.Type = b[6]
	if dt == timeOverflow {
		if b, err = r.Peek(4); err != nil {
			return
		}
		dt = int64(be.Uint32(b))
	}
	cr.Time += dt
	return
}

func (cr *chunkReader) readDeltaHeader(r *reader) (err error) {
	var b []byte
	if b, err = r.Peek(3); err != nil {
		return
	}
	dt := int64(getUint24(b))
	cr.pos = 0
	if dt == timeOverflow {
		if b, err = r.Peek(4); err != nil {
			return
		}
		dt = int64(be.Uint32(b))
	}
	cr.Time += dt
	return
}

type writer struct {
	mu   sync.Mutex
	w    io.Writer
	buf  []byte
	pos  int
	ack  int
	size int
}

func newWriter(w io.Writer) *writer {
	return &writer{
		w:    w,
		size: 128,
	}
}

func (w *writer) next(n int) (b []byte) {
	p := w.pos + n
	if len(w.buf) < p {
		w.grow(n << 1)
	}
	b, w.pos = w.buf[w.pos:p], p
	return
}

func (w *writer) grow(n int) {
	p := w.pos + n
	if len(w.buf) < p {
		buf := make([]byte, (1+(p>>10))<<10)
		if w.pos > 0 {
			copy(buf, w.buf[:w.pos])
		}
		w.buf = buf
	}
}

func (w *writer) WriteFull(id uint32, ts int64, ct uint8, str uint32, data []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := len(data)
	w.grow(n + 18 + 3*int(n/w.size))
	w.writeHeader(fmtFull, id)
	w.writeFullHeader(ts, ct, str, n)
	for n > 0 {
		if n > w.size {
			n -= copy(w.next(w.size), data)
			data = data[w.size:]
			w.writeHeader(fmtData, id)
		} else {
			copy(w.next(n), data)
			return
		}
	}
}

func (w *writer) writeHeader(fmt uint8, id uint32) {
	if fmt <<= 6; id < 64 {
		b := w.next(1)
		b[0] = fmt | uint8(id)
	} else if id -= 64; id <= 0xff {
		b := w.next(2)
		b[0] = fmt
		b[1] = uint8(id)
	} else {
		b := w.next(3)
		b[0] = fmt | 1
		be.PutUint16(b[1:], uint16(id))
	}
}

func (w *writer) writeFullHeader(ts int64, ct uint8, str uint32, n int) {
	var b []byte
	if ts < timeOverflow {
		b = w.next(11)
		putUint24(b, uint32(ts))
	} else {
		b = w.next(15)
		putUint24(b, uint32(timeOverflow))
		be.PutUint32(b[11:], uint32(ts))
	}
	putUint24(b[3:], uint32(n))
	b[6] = ct
	le.PutUint32(b[7:], str)
}

func (w *writer) writeStreamHeader(dt int64, ct uint8, n int) {
	var b []byte
	if dt < timeOverflow {
		b = w.next(7)
		putUint24(b, uint32(dt))
	} else {
		b = w.next(11)
		putUint24(b, uint32(timeOverflow))
		be.PutUint32(b[7:], uint32(dt))
	}
	putUint24(b[3:], uint32(n))
	b[6] = ct
}

func (w *writer) writeDeltaHeader(dt int64) {
	if dt < timeOverflow {
		b := w.next(3)
		putUint24(b, uint32(dt))
	} else {
		b := w.next(7)
		putUint24(b, uint32(timeOverflow))
		be.PutUint32(b[4:], uint32(dt))
	}
}

func (w *writer) Flush() (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.pos > 0 {
		_, err = w.w.Write(w.buf[:w.pos])
		w.pos = 0
	}
	return
}
