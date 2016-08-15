package amf

import (
	"bufio"
	"errors"
	"io"
	"reflect"
	"strconv"
)

const (
	typeNumber    = uint8(0x00)
	typeBoolean   = uint8(0x01)
	typeString    = uint8(0x02)
	typeObject    = uint8(0x03)
	typeNull      = uint8(0x05)
	typeUndefined = uint8(0x06)
	typeReference = uint8(0x07)
	typeArray     = uint8(0x08)
	typeObjectEnd = uint8(0x09)
	typeVector    = uint8(0x0a)
	typeDate      = uint8(0x0b)
	typeStringExt = uint8(0x0c)
	typeXml       = uint8(0x0f)
	typeInstance  = uint8(0x10)
)

type Decoder struct {
	r    io.Reader
	buf  *bufio.Reader
	skip int
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:   r,
		buf: bufio.NewReader(r),
	}
}

func (dec *Decoder) next(n int) ([]byte, error) {
	if dec.skip > 0 {
		if _, err := dec.buf.Discard(dec.skip); err != nil {
			return nil, err
		}
	}
	dec.skip = n
	return dec.buf.Peek(n)
}

func (dec *Decoder) Decode(v interface{}) error {
	if v == nil {
		return errors.New("amf: decode nil")
	}
	// TODO: unmarshaler
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return errors.New("amf: decode not a pointer")
	}
	return dec.decodeReflect(rv.Elem())
}

func (dec *Decoder) decodeReflect(v reflect.Value) error {
	switch v.Kind() {
	/*	case reflect.Bool:
			enc.EncodeBool(v.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			enc.EncodeNumber(float64(v.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			enc.EncodeNumber(float64(v.Uint()))
		case reflect.String:
			enc.EncodeString(v.String())
		case reflect.Struct:
			return enc.encodeStruct(v)
		case reflect.Ptr:
			return enc.encodeReflect(v.Elem())*/
	//default:
	//return errUnknownType{v.Type()}
	}
	return nil
}

func (dec *Decoder) DecodeNext() (v interface{}, err error) {
	var b []byte
	if b, err = dec.next(1); err != nil {
		return
	}
	switch b[0] {
	case typeNumber:
		if b, err = dec.next(8); err != nil {
			return
		}
		return getFloat64(b), nil
	case typeBoolean:
		if b, err = dec.next(1); err != nil {
			return
		}
		return b[0] != 0, nil
	case typeString:
		if b, err = dec.next(2); err != nil {
			return
		} else if b, err = dec.next(int(be.Uint16(b))); err != nil {
			return
		}
		return string(b), nil
	case typeArray:
		if b, err = dec.next(4); err != nil {
			return
		}
		fallthrough
	case typeObject:
		var it interface{}
		m := make(map[string]interface{})
		for {
			if b, err = dec.next(2); err != nil {
				return
			}
			n := int(be.Uint16(b))
			if n == 0 {
				break
			}
			if b, err = dec.next(n); err != nil {
				return
			}
			name := string(b)
			if it, err = dec.DecodeNext(); err != nil {
				return
			}
			m[name] = it
		}
		if b, err = dec.next(1); err != nil {
			return
		}
		return m, nil
	case typeNull, typeUndefined:
		return nil, nil
	case typeStringExt:
		if b, err = dec.next(4); err != nil {
			return
		} else if b, err = dec.next(int(be.Uint32(b))); err != nil {
			return
		}
		return string(b), nil
	default:
		return nil, errUnsupportedMarker{b[0]}
	}
}

type errUnsupportedMarker struct {
	t uint8
}

func (err errUnsupportedMarker) Error() string {
	return "amf: unsupported marker: " + strconv.Itoa(int(err.t))
}
