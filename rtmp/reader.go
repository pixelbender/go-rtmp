package rtmp

import (
	"bufio"
	"io"
	"log"
)

const (
	fmtFull   = uint8(0x00)
	fmtHeader = uint8(0x01)
	fmtDelta  = uint8(0x02)
	fmtNone   = uint8(0x03)
)

const timeOverflow = int64(0xffffff)

type chunk struct {
	Id     uint32
	Time   int64
	Type   uint8
	Stream uint32
	Data   []byte
}

type reader struct {
	r    io.Reader
	buf  *bufio.Reader
	mux  map[uint32]*chunkReader
	skip int
	size int
}

func newReader(r io.Reader) *reader {
	return &reader{
		r:    r,
		buf:  bufio.NewReader(r),
		mux:  make(map[uint32]*chunkReader),
		size: 128,
	}
}

func (r *reader) Read(b []byte) (n int, err error) {
	r.discard()
	return r.buf.Read(b)
}

func (r *reader) Peek(n int) (b []byte, err error) {
	r.discard()
	r.skip = n
	return r.buf.Peek(n)
}

func (r *reader) ReadByte() (b byte, err error) {
	r.discard()
	return r.buf.ReadByte()
}

func (r *reader) discard() {
	if r.skip > 0 {
		r.buf.Discard(r.skip)
		r.skip = 0
	}
}

func (r *reader) readChunkHeader() (fmt uint8, id uint32, err error) {
	var b byte
	if b, err = r.ReadByte(); err != nil {
		return
	}
	log.Printf("!!!! %v", b)
	fmt, id = b>>6&0x03, uint32(b&0x3f)
	if id == 0 {
		if b, err = r.ReadByte(); err != nil {
			return
		}
		id = uint32(b) + 64
	} else if id == 1 {
		if b, err = r.ReadByte(); err != nil {
			return
		}
		id = uint32(b)
		if b, err = r.ReadByte(); err != nil {
			return
		}
		id = id<<8 + uint32(b) + 64
	}
	return
}

func (r *reader) ReadChunk() (result *chunk, err error) {
	fmt, id, err := r.readChunkHeader()
	if err != nil {
		return nil, err
	}
	cr, ok := r.mux[id]
	if !ok {
		cr = new(chunkReader)
		cr.Id = id
		r.mux[id] = cr
	}
	var done bool
	if done, err = cr.fill(r, fmt, r.size); err != nil {
		return
	}
	if done {
		result = &cr.chunk
	}
	return
}

type chunkReader struct {
	chunk
	buf []byte
	pos int
	len int
}

func (cr *chunkReader) setLength(n int) {
	if len(cr.buf) < n {
		cr.buf = make([]byte, (1+(n>>8))<<8)
	}
	cr.len, cr.pos = n, 0
}

func (cr *chunkReader) fill(r *reader, fmt uint8, size int) (done bool, err error) {
	var b []byte
	switch fmt {
	case fmtFull:
		if b, err = r.Peek(11); err != nil {
			return
		}
		cr.setLength(int(getUint24(b[3:])))
		cr.Type = b[6]
		cr.Stream = getStream(b[7:])
		ts := int64(getUint24(b))
		if ts == timeOverflow {
			if b, err = r.Peek(4); err != nil {
				return
			}
			ts = int64(getUint32(b))
		}
		cr.Time = ts
	case fmtHeader:
		if b, err = r.Peek(7); err != nil {
			return
		}
		cr.setLength(int(getUint24(b[3:])))
		cr.Type = b[6]
		fallthrough
	case fmtDelta:
		dt := int64(getUint24(b))
		if dt == timeOverflow {
			if b, err = r.Peek(4); err != nil {
				return
			}
			dt = int64(getUint32(b))
		}
		cr.Time += dt
	}
	n := cr.len - cr.pos
	if n > size {
		n = size
	} else if cr.pos == 0 {
		if cr.Data, err = r.Peek(cr.len); err != nil {
			return
		}
		return true, nil
	}
	off := cr.pos + n
	if _, err = r.Read(cr.buf[cr.pos:off]); err != nil {
		return
	}
	if off == cr.len {
		cr.pos, cr.Data = 0, cr.buf[:off]
		return true, nil
	} else {
		cr.pos = off
	}
	return
}
