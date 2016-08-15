package rtmp

import (
	"io"
)

type writer struct {
	w    io.Writer
	buf  []byte
	pos  int
	mux  map[uint32]*chunkWriter
	ack  int
	size int
}

func newWriter(w io.Writer) *writer {
	return &writer{
		w:    w,
		mux:  make(map[uint32]*chunkWriter),
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

func (w *writer) writeChunkHeader(fmt uint8, id uint32) {
	var b []byte
	if fmt <<= 6; id < 64 {
		b = w.next(1)
		b[0] = fmt | uint8(id)
	} else if id -= 64; id <= 0xff {
		b = w.next(2)
		b[0] = fmt
		b[1] = uint8(id)
	} else {
		b = w.next(3)
		b[0] = fmt | 1
		putUint16(b[1:], uint16(id))
	}
}

func (w *writer) writeFullHeader(ts int64, ct uint8, str uint32, n int) {
	b := w.next(11)
	if ts < timeOverflow {
		putUint24(b, uint32(ts))
	} else {
		putUint24(b, uint32(timeOverflow))
		putUint32(w.next(4), uint32(ts))
	}
	putUint24(b[3:], uint32(n))
	b[6] = ct
	putStream(b[7:], str)
}

func (w *writer) WriteFull(id uint32, ts int64, ct uint8, str uint32, data []byte) {
	n := len(data)

	w.grow(n + 18 + 3*int(n/w.size))
	w.writeChunkHeader(fmtFull, id)
	w.writeFullHeader(ts, ct, str, n)

	for n > 0 {
		if n > w.size {
			n -= copy(w.next(w.size), data)
			data = data[w.size:]
			w.writeChunkHeader(fmtNone, id)
		} else {
			copy(w.next(n), data)
			break
		}
	}
}

func (w *writer) Flush() (err error) {
	if w.pos > 0 {
		_, err = w.w.Write(w.buf[:w.pos])
		w.pos = 0
	}
	return
}

type chunkWriter struct {
	buf []byte
	pos int
	len int
}
