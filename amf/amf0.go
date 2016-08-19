package amf

import (
	"encoding/binary"
	"math"
	"reflect"
	"strconv"
)

const (
	amf0Number      = uint8(0x00) // float64
	amf0Boolean     = uint8(0x01) // bool
	amf0String      = uint8(0x02) // string
	amf0Object      = uint8(0x03) // map[string]interface{}
	amf0Null        = uint8(0x05) // nil
	amf0Undefined   = uint8(0x06) // nil
	amf0Reference   = uint8(0x07) // pointer
	amf0Array       = uint8(0x08) // []interface{} or map[string]interface{}
	amf0ObjectEnd   = uint8(0x09)
	amf0StrictArray = uint8(0x0a) // []interface{}
	amf0Date        = uint8(0x0b) // time.Time
	amf0StringExt   = uint8(0x0c) // stirng
	amf0Xml         = uint8(0x0f) // string
	amf0Instance    = uint8(0x10) // map[string]interface{}
)

type amf0Encoder struct {
	writer
}

func (enc *amf0Encoder) Encode(v interface{}) error {
	if m, ok := v.(Marshaler); ok {
		return m.MarshalAMF(enc)
	}
	return enc.encodeValue(reflect.ValueOf(v))
}

func (enc *amf0Encoder) encodeValue(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Invalid:
		enc.EncodeNull()
	case reflect.Bool:
		enc.EncodeBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		enc.EncodeInt(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		enc.EncodeUint(v.Uint())
	case reflect.Float32, reflect.Float64:
		enc.EncodeFloat(v.Float())
	case reflect.String:
		enc.EncodeString(v.String())
	case reflect.Slice, reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			enc.EncodeBytes(v.Bytes())
		default:
			return enc.encodeArray(v)
		}
	case reflect.Struct:
		return enc.encodeStruct(v)
	case reflect.Map:
		return enc.encodeMap(v)
	case reflect.Ptr:
		return enc.encodeValue(v.Elem())
	default:
		return errUnsType(v.Kind())
	}
	return nil
}

func (enc *amf0Encoder) EncodeNull() {
	enc.Next(1)[0] = amf0Null
}

func (enc *amf0Encoder) EncodeBool(v bool) {
	b := enc.Next(2)
	b[0] = amf0Boolean
	if v {
		b[1] = 1
	} else {
		b[1] = 0
	}
}

func (enc *amf0Encoder) EncodeInt(v int64) {
	enc.EncodeFloat(float64(v))
}

func (enc *amf0Encoder) EncodeUint(v uint64) {
	enc.EncodeFloat(float64(v))
}

func (enc *amf0Encoder) EncodeFloat(v float64) {
	b := enc.Next(9)
	b[0] = amf0Number
	putFloat64(b[1:], v)
}

func (enc *amf0Encoder) EncodeString(v string) {
	// TODO: reference map
	copy(enc.allocString(len(v)), v)
}

func (enc *amf0Encoder) EncodeBytes(v []byte) {
	// TODO: reference map
	copy(enc.allocString(len(v)), v)
}

func (enc *amf0Encoder) allocString(n int) []byte {
	if n > 0xffff {
		b := enc.Next(n + 5)
		b[0] = amf0StringExt
		be.PutUint32(b[1:], uint32(n))
		return b[5:]
	}
	b := enc.Next(n + 3)
	b[0] = amf0String
	be.PutUint16(b[1:], uint16(n))
	return b[3:]
}

func (enc *amf0Encoder) encodeString(v string) {
	n := len(v)
	b := enc.Next(n + 2)
	be.PutUint32(b, uint32(n))
	copy(b[2:], v)
}

func (enc *amf0Encoder) encodeArray(v reflect.Value) (err error) {
	// TODO: reference map
	b := enc.Next(5)
	b[0] = amf0Array
	n := v.Len()
	be.PutUint32(b[1:], uint32(n))
	for i := 0; i < n; i++ {
		enc.encodeString(strconv.Itoa(i))
		if err = enc.encodeValue(v.Index(i)); err != nil {
			return
		}
	}
	putUint24(enc.Next(3), uint32(amf0ObjectEnd))
	return
}

func (enc *amf0Encoder) encodeStruct(v reflect.Value) (err error) {
	// TODO: reference map
	m := getStructMapping(v.Type())
	b := enc.Next(1)
	b[0] = amf0Object
	for _, f := range m.fields {
		r := v.Field(f.index)
		if f.opt && isEmptyValue(r) {
			continue
		}
		enc.encodeString(f.name)
		if err = enc.encodeValue(r); err != nil {
			return
		}
	}
	putUint24(enc.Next(3), uint32(amf0ObjectEnd))
	return
}

func (enc *amf0Encoder) encodeMap(v reflect.Value) (err error) {
	// TODO: reference map
	b := enc.Next(1)
	b[0] = amf0Object
	for _, k := range v.MapKeys() {
		switch k.Kind() {
		case reflect.String:
			if s := k.String(); s != "" {
				enc.encodeString(s)
				if err = enc.encodeValue(v.MapIndex(k)); err != nil {
					return
				}
			}
		default:
			return errUnsKeyType(k.Kind())
		}
	}
	putUint24(enc.Next(3), uint32(amf0ObjectEnd))
	return
}

type amf0Decoder struct {
	Reader
	b   []byte
	err error
}

func (dec *amf0Decoder) next(n int) bool {
	dec.b, dec.err = dec.Next(n)
	return dec.err == nil
}

func (dec *amf0Decoder) Decode(v interface{}) error {
	if v == nil {
		return errDecodeNil
	}
	if m, ok := v.(Unmarshaler); ok {
		return m.UnmarshalAMF(dec)
	}
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Ptr {
		return errDecodeNotPtr
	}
	return dec.decodeValue(v)
}

func (dec *amf0Decoder) decodeValue(v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		r, err := dec.DecodeBool()
		if err != nil {
			return err
		}
		v.SetBool(r)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		r, err := dec.DecodeInt()
		if err != nil {
			return err
		}
		v.SetInt(r)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		r, err := dec.DecodeUint()
		if err != nil {
			return err
		}
		v.SetUint(r)
	case reflect.Float32, reflect.Float64:
		r, err := dec.DecodeFloat()
		if err != nil {
			return err
		}
		v.SetFloat(r)
	case reflect.String:
		r, err := dec.DecodeString()
		if err != nil {
			return err
		}
		v.SetString(r)
	case reflect.Slice, reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			enc.EncodeBytes(v.Bytes())
		default:
			return enc.encodeArray(v)
		}
	case reflect.Struct:
		enc.encodeStruct(v)
	case reflect.Map:
		enc.encodeMap(v)
	case reflect.Ptr:
		return enc.EncodeValue(v.Elem())
	default:
		return errUnsType(v.Kind())
	}
	return nil
}

func (dec *amf0Decoder) DecodeBool() (bool, error) {
	if dec.next(1) {
		switch dec.b[0] {
		case amf0Boolean:
			return dec.decodeBool()
		default:
			false, &errUnsDecodeType{dec.b[0], reflect.Bool}
		}
	}
	return false, dec.err
}

func (dec *amf0Decoder) DecodeInt() (int64, error) {
	if dec.next(1) {
		switch dec.b[0] {
		case amf0Number:
			return dec.decodeBool()
		default:
			false, &errUnsDecodeType{dec.b[0], reflect.Bool}
		}
	}
	return false, dec.err
}

func (dec *amf0Decoder) DecodeFloat() (float64, error) {
	if dec.next(1) {
		switch dec.b[0] {
		case amf0Number:
			return dec.decodeBool()
		default:
			false, &errUnsDecodeType{dec.b[0], reflect.Bool}
		}
	}
	return false, dec.err
}

func (dec *amf0Decoder) DecodeNext() (v interface{}, err error) {
	if !dec.next(1) {
		return nil, dec.err
	}
	switch dec.b[0] {
	case amf0Number:
		if dec.next(8) {
			return getFloat64(dec.b), nil
		}
	case amf0Boolean:
		if dec.next(1) {
			return dec.b[0] != 0, nil
		}
	case amf0String:
		return dec.decodeString(false)
	case amf0Object:
		return dec.decodeObject()
	case amf0Null, amf0Undefined:
		return nil, nil
	//case amf0Reference:
	// TODO: add support for refs
	case amf0Array:
		if _, err = dec.Next(4); err != nil {
			return
		}
		return dec.decodeObject()
	case amf0StrictArray:
		return dec.decodeArray()
	//case amf0Date:
	// TODO: add support for time
	//	return dec.decodeTime()
	case amf0StringExt, amf0Xml:
		return dec.decodeString(true)
	case amf0Instance:
		// TODO: class-name support
		dec.decodeString(false)
		return dec.decodeObject()
	default:
		dec.err = errUnsMarker(dec.b[0])
	}
	if dec.err != nil {
		return nil, dec.err
	}
	return
}

func (dec *amf0Decoder) Skip() error {
	if dec.next(1) && dec.skipMarker(dec.b[0]) {
		return nil
	}
	return dec.err
}

func (dec *amf0Decoder) skipMarker(m uint8) bool {
	switch m {
	case amf0Number:
		return dec.next(8)
	case amf0Boolean:
		return dec.next(1)
	case amf0String:
		return dec.skipString(false)
	case amf0Object:
		return dec.skipObject()
	case amf0Null, amf0Undefined:
		return true
	case amf0Reference:
		return dec.next(2)
	case amf0Array:
		return dec.next(2) && dec.skipObject()
	case amf0StrictArray:
		return dec.skipArray()
	case amf0Date:
		return dec.next(10)
	case amf0StringExt, amf0Xml:
		return dec.skipString(true)
	case amf0Instance:
		return dec.skipString(false) && dec.skipObject()
	default:
		dec.err = errUnsMarker(m)
	}
	return false
}

func (dec *amf0Decoder) skipString(ext bool) bool {
	if ext {
		if dec.next(4) {
			n := int(be.Uint32(dec.b))
			return dec.next(n)
		}
	} else {
		if dec.next(2) {
			n := int(be.Uint16(dec.b))
			return dec.next(n)
		}
	}
	return false
}

func (dec *amf0Decoder) skipObject() bool {
	for dec.next(2) {
		n := int(be.Uint16(dec.b))
		if dec.next(n) && n > 0 && dec.Skip() == nil {
			continue
		}
		break
	}
	return dec.next(1)
}

func (dec *amf0Decoder) skipArray() bool {
	if dec.next(4) {
		n := int(be.Uint32(dec.b))
		for n > 0 && dec.Skip() == nil {
			n--
		}
		return n == 0
	}
	return false
}

func (dec *amf0Decoder) decodeBool() (bool, error) {
	if dec.next(1) {
		return dec.b[0] != 0, nil
	}
	return false, dec.err
}

func (dec *amf0Decoder) decodeString(ext bool) (string, error) {
	if ext {
		if dec.next(4) {
			if n := int(be.Uint32(dec.b)); dec.next(n) {
				return string(dec.b), nil
			}
		}
	} else {
		if dec.next(2) {
			if n := int(be.Uint16(dec.b)); dec.next(n) {
				return string(dec.b), nil
			}
		}
	}
	return "", dec.err
}

func (dec *amf0Decoder) decodeObject() (map[string]interface{}, error) {
	r := make(map[string]interface{})
	for {
		name, err := dec.decodeString(false)
		if err != nil {
			return nil, err
		}
		if name == "" {
			break
		}
		if r[name], err = dec.DecodeNext(); err != nil {
			return nil, err
		}
	}
	if dec.next(1) {
		return r, nil
	}
	return nil, dec.err
}

func (dec *amf0Decoder) decodeArray() ([]interface{}, error) {
	if !dec.next(4) {
		return nil, dec.err
	}
	n := int(be.Uint32(dec.b))
	cap := n
	if cap > 1024 {
		cap = 1024
	}
	r := make([]interface{}, 0, cap)
	for i := 0; i < n; i++ {
		v, err := dec.DecodeNext()
		if err != nil {
			return nil, err
		}
		r = append(r, v)
	}
	return r, nil
}

func getFloat64(b []byte) float64 {
	return math.Float64frombits(be.Uint64(b))
}

func putFloat64(b []byte, v float64) {
	be.PutUint64(b, math.Float64bits(v))
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}

var be = binary.BigEndian
