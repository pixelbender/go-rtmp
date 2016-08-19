package amf

import (
	"bufio"
	"io"
)

type Unmarshaler interface {
	UnmarshalAMF(dec Decoder) error
}

type Reader interface {
	Read(b []byte) (int, error)
	Next(n int) ([]byte, error)
}

type Decoder interface {
	Reader
	Decode(v interface{}) error
	//DecodeValue(v reflect.Value) error
	Skip() error
	DecodeNext() (interface{}, error)

	//DecodeBool(v bool)
	//DecodeInt(v int64)
	//DecodeUint(v uint64)
	//DecodeFloat(v float64)
	//DecodeString(v string)
	//DecodeBytes(v []byte)
}

func NewDecoder(ver uint8, r io.Reader) Decoder {
	return newDecoder(ver, &peeker{Reader: bufio.NewReader(r)})
}

func NewDecoderBytes(ver uint8, b []byte) Decoder {
	return newDecoder(ver, &reader{buf: b})
}

func newDecoder(ver uint8, r Reader) Decoder {
	if ver == 3 {
		return &amf3Decoder{Reader: r}
	}
	return &amf0Decoder{Reader: r}
}

type peeker struct {
	*bufio.Reader
	skip int
}

func (p *peeker) Read(b []byte) (int, error) {
	if p.skip > 0 {
		p.Discard(p.skip)
		p.skip = 0
	}
	return p.Reader.Read(b)
}

func (p *peeker) Next(n int) ([]byte, error) {
	if p.skip > 0 {
		p.Discard(p.skip)
	}
	p.skip = n
	return p.Reader.Peek(n)
}

type reader struct {
	buf []byte
	pos int
}

func (r *reader) Read(b []byte) (n int, err error) {
	n = copy(b, r.buf[r.pos:])
	if n < len(b) {
		err = io.EOF
		r.pos = len(r.buf)
	} else {
		r.pos += n
	}
	return
}

func (r *reader) Next(n int) ([]byte, error) {
	off := r.pos + n
	if len(r.buf) < off {
		return r.buf[r.pos:], io.EOF
	}
	b := r.buf[r.pos:off]
	r.pos = off
	return b, nil
}
