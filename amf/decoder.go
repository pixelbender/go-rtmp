package amf

import (
	"bufio"
	"io"
)

type Unmarshaler interface {
	UnmarshalAMF(dec Decoder) error
}

type reader interface {
	Peek(n int) ([]byte, error)
}

type Decoder interface {
	reader
	Decode(v interface{}) error
	//DecodeValue(v reflect.Value) error
	Skip() error
	DecodeInt() (int64, error)
	DecodeUint() (uint64, error)
	DecodeFloat() (float64, error)
	DecodeString() (string, error)
	DecodeBytes() ([]byte, error)
	//DecodeBool(v bool)
	//DecodeInt(v int64)
	//DecodeUint(v uint64)
	//DecodeFloat(v float64)
	//DecodeString(v string)
	//DecodeBytes(v []byte)
}

func NewDecoder(ver uint8, r io.Reader) Decoder {
	return newDecoder(ver, &bufReader{bufio.NewReader(r), 0})
}

func NewDecoderBytes(ver uint8, b []byte) Decoder {
	return newDecoder(ver, &bytesReader{b, 0})
}

func newDecoder(ver uint8, r reader) Decoder {
	if ver == 3 {
		return &amf3Decoder{reader: r}
	}
	return &amf0Decoder{reader: r}
}

type bufReader struct {
	*bufio.Reader
	skip int
}

func (r *bufReader) Peek(n int) ([]byte, error) {
	if r.skip > 0 {
		r.Discard(r.skip)
	}
	r.skip = n
	return r.Reader.Peek(n)
}

type bytesReader struct {
	buf []byte
	pos int
}

func (r *bytesReader) Peek(n int) ([]byte, error) {
	off := r.pos + n
	if len(r.buf) < off {
		return r.buf[r.pos:], io.EOF
	}
	b := r.buf[r.pos:off]
	r.pos = off
	return b, nil
}
