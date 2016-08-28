package amf

import (
	"time"
)

// Marshaler is the interface implemented by objects that can marshal themselves into valid AMF.
type Marshaler interface {
	MarshalAMF(w *Writer) error
}

type Encoder interface {
	Encode(v interface{}) error
	WriteNull()
	WriteBool(v bool)
	WriteInt(v int64)
	WriteUint(v uint64)
	WriteFloat(v float64)
	WriteString(v string)
	WriteBytes(v []byte)
	WriteTime(v time.Time)
	Reset()
	Next(n int) []byte
	Bytes() []byte
}

func NewEncoder(ver uint8) Encoder {
	w := &Writer{}
	//if ver == 3 {
	//	return &amf3Encoder{writer:w}
	//}
	return &amf0Encoder{Writer: w}
}

type Writer struct {
	buf []byte
	pos int
}

func (w *Writer) Reset() {
	w.pos = 0
}

func (w *Writer) Next(n int) (b []byte) {
	p := w.pos + n
	if len(w.buf) < p {
		b := make([]byte, (1+((p-1)>>10))<<10)
		if w.pos > 0 {
			copy(b, w.buf[:w.pos])
		}
		w.buf = b
	}
	b, w.pos = w.buf[w.pos:p], p
	return
}

func (w *Writer) Bytes() []byte {
	return w.buf[:w.pos]
}
