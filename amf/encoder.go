package amf

import (
	"encoding/binary"
	"log"
	"math"
	"reflect"
	"strings"
)

type Marshaler struct {
	//MarshalAMF()
}

type Encoder struct {
	buf []byte
	pos int
}

func (enc *Encoder) next(n int) (b []byte) {
	p := enc.pos + n
	if len(enc.buf) < p {
		enc.grow(n << 1)
	}
	b, enc.pos = enc.buf[enc.pos:p], p
	return
}

func (enc *Encoder) grow(n int) {
	p := enc.pos + n
	if len(enc.buf) < p {
		buf := make([]byte, (1+(p>>10))<<10)
		if enc.pos > 0 {
			copy(buf, enc.buf[:enc.pos])
		}
		enc.buf = buf
	}
}

func (enc *Encoder) Reset() {
	enc.pos = 0
}

func (enc *Encoder) Bytes() []byte {
	return enc.buf[:enc.pos]
}

func (end *Encoder) Encode(v interface{}) error {
	var b []byte
	if v == nil {
		b = end.next(1)
		b[0] = typeNull
		return nil
	}
	// TODO: cusrom marshaler
	return end.encodeReflect(reflect.ValueOf(v))
}

func (enc *Encoder) encodeReflect(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
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
		return enc.encodeReflect(v.Elem())
	default:
		return errUnsupportedType{v.Type()}
	}
	return nil
}

func (enc *Encoder) EncodeBool(v bool) {
	b := enc.next(2)
	b[0] = typeBoolean
	if v {
		b[1] = 1
	} else {
		b[1] = 0
	}
}

func (enc *Encoder) EncodeNumber(v float64) {
	b := enc.next(9)
	b[0] = typeNumber
	putFloat64(b[1:], v)
}

func (enc *Encoder) EncodeString(v string) {
	if n := len(v); n > 0xffff {
		b := enc.next(n + 5)
		b[0] = typeStringExt
		be.PutUint32(b[1:], uint32(n))
		copy(b[5:], v)
	} else {
		b := enc.next(n + 3)
		b[0] = typeString
		be.PutUint16(b[1:], uint16(n))
		copy(b[3:], v)
	}
}

func (enc *Encoder) encodeStruct(v reflect.Value) (err error) {
	t := v.Type()
	n := t.NumField()

	b := enc.next(1)
	b[0] = typeObject

	for i := 0; i < n; i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("amf"), ",")

		name := tag[0]
		if name == "-" || name == "" {
			continue
		}
		b = enc.next(len(name) + 2)
		be.PutUint16(b, uint16(len(name)))
		copy(b[2:], name)
		log.Printf(">> %v = %v", name, v.Field(i).Type())
		if err = enc.encodeReflect(v.Field(i)); err != nil {
			return
		}

	}
	b = enc.next(3)
	putUint24(b, uint32(typeObjectEnd))
	return
}

type errUnsupportedType struct {
	t reflect.Type
}

func (err errUnsupportedType) Error() string {
	return "amf: unknown type: " + err.t.String()
}

var be = binary.BigEndian

func getFloat64(b []byte) float64 {
	return math.Float64frombits(be.Uint64(b))
}

func getFloat32(b []byte) float32 {
	return math.Float32frombits(be.Uint32(b))
}

func putFloat64(b []byte, v float64) {
	be.PutUint64(b, math.Float64bits(v))
}

func putFloat32(b []byte, v float32) {
	be.PutUint32(b, math.Float32bits(v))
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
