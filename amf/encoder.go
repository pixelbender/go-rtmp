package amf

// Marshaler is the interface implemented by objects that can marshal themselves into valid AMF.
type Marshaler interface {
	MarshalAMF(enc Encoder) error
}

type Writer interface {
	Write(b []byte) (int, error)
	Next(n int) []byte
}

type Encoder interface {
	Writer
	Encode(v interface{}) error
	EncodeNull()
	EncodeBool(v bool)
	EncodeInt(v int64)
	EncodeUint(v uint64)
	EncodeFloat(v float64)
	EncodeString(v string)
	EncodeBytes(v []byte)
	Bytes() []byte
}

func NewEncoder(ver uint8) Encoder {
	if ver == 3 {
		return &amf3Encoder{}
	}
	return &amf0Encoder{}
}

type writer struct {
	buf []byte
	pos int
}

func (w *writer) Write(b []byte) (int, error) {
	return copy(w.Next(len(b)), b), nil
}

func (w *writer) Next(n int) (b []byte) {
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

func (w *writer) Bytes() []byte {
	return w.buf[:w.pos]
}
