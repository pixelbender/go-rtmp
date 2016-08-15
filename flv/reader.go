package flv

import (
	"bufio"
	"errors"
	"io"
)

// ErrFormat is returned by Reader when a stream is not a valid FLV data.
var ErrFormat = errors.New("flv: incorrect format")

// Reader reads FLV header and tags from an input stream.
type Reader struct {
	buf    *bufio.Reader
	seeker io.ReadSeeker
	r      io.Reader
	skip   *io.LimitedReader
	header *Header
	tag    *Tag
}

// NewReader returns a new reader that reads from r.
func NewReader(r io.Reader) *Reader {
	seeker, _ := r.(io.ReadSeeker)
	buf, _ := r.(*bufio.Reader)
	if buf != nil {
		buf = bufio.NewReader(r)
	}
	return &Reader{buf: buf, seeker: seeker, r: r}
}

// Read reads FLV header
func (r *Reader) ReadHeader() (h *Header, err error) {
	if h = r.header; h != nil {
		return
	}
	var b []byte
	if b, err = r.buf.Peek(13); err != nil {
		return
	}
	h = &Header{
		Signature: getUint24(b[0:]),
		Version:   b[3],
		Flags:     b[4],
	}
	off := getUint32(b[5:]) - 9
	if h.Signature != sign || h.Version != 1 || off < 0 {
		return nil, ErrFormat
	}
	if _, err = r.buf.Discard(15); err != nil {
		return
	}
	if off > 0 {
		r.skip = io.LimitReader(r.buf, off)
	}
	return
}

// Read reads FLV tag and returns data reader.
// Data reader is not valid after next Read.
func (r *Reader) ReadTag() (tag *Tag, data io.Reader, err error) {
	if r.header == nil {
		if _, err = r.ReadHeader(); err != nil {
			return
		}
	}
	r.Skip()
	var b []byte
	if b, err = r.buf.Peek(15); err != nil {
		return
	}
	if _, err = r.buf.Discard(15); err != nil {
		return
	}
	if p := getUint32(b); r.tag != nil && r.tag.Size+11 != p {
		err = ErrFormat
		return
	}
	tag = &Tag{
		Type:   b[4],
		Size:   getInt24(b[5:]),
		Time:   getTime(b[8:]),
		Stream: getUint24(b[12:]),
	}
	data = io.LimitReader(r.buf, tag.Size)
	r.tag, r.skip = tag, data
	return
}

// Skip skips unread data before next FLV tag
func (r *Reader) Skip() (err error) {
	if r.skip == nil {
		return
	}
	n := r.skip.N
	if n > 0 {
		b := int64(r.buf.Buffered())
		if b < n && r.seeker != nil {
			_, err = r.seeker.Seek(n-b, 1)
			r.buf.Reset(r.r)
		} else {
			_, err = r.buf.Discard(n)
		}
	}
	return
}

func getInt24(b []byte) int {
	return int(b[2]) | int(b[1])<<8 | int(b[0])<<16
}

func getUint24(b []byte) uint32 {
	return uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
}

func getTime(b []byte) int64 {
	return int64(getUint24(b) | uint32(b[3])<<24)
}

func getUint32(b []byte) uint32 {
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}
