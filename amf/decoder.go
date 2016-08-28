package amf

import (
	"io"
	"time"
)

type Unmarshaler interface {
	UnmarshalAMF(r *Reader) error
}

type Decoder interface {
	Decode(v interface{}) error
	Skip() error
	ReadBool() (bool, error)
	ReadInt() (int64, error)
	ReadUint() (uint64, error)
	ReadFloat() (float64, error)
	ReadString() (string, error)
	ReadBytes() ([]byte, error)
	ReadTime() (time.Time, error)
}

func NewDecoder(ver uint8, v []byte) Decoder {
	r := &Reader{buf: v}
	//if ver == 3 {
	//	return &amf3Decoder{Reader: r}
	//}
	return &amf0Decoder{Reader: r}
}

type Reader struct {
	buf []byte
	pos int
}

func (r *Reader) Next(n int) ([]byte, error) {
	p := r.pos + n
	if len(r.buf) < r.pos+n {
		return nil, io.EOF
	}
	off := r.pos
	r.pos = p
	return r.buf[off : off+n], nil
}
