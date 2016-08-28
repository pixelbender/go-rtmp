package amf

import (
	"encoding/binary"
	"math"
	"reflect"
	"time"
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
	*Writer
}

func (enc *amf0Encoder) Encode(v interface{}) error {
	if m, ok := v.(Marshaler); ok {
		return m.MarshalAMF(enc.Writer)
	}
	return encodeValue(reflect.ValueOf(v), enc)
}

func (enc *amf0Encoder) WriteNull() {
	enc.Next(1)[0] = amf0Null
}

func (enc *amf0Encoder) WriteBool(v bool) {
	b := enc.Next(2)
	b[0] = amf0Boolean
	if v {
		b[1] = 1
	} else {
		b[1] = 0
	}
}

func (enc *amf0Encoder) WriteInt(v int64) {
	enc.WriteFloat(float64(v))
}

func (enc *amf0Encoder) WriteUint(v uint64) {
	enc.WriteFloat(float64(v))
}

func (enc *amf0Encoder) WriteFloat(v float64) {
	b := enc.Next(9)
	b[0] = amf0Number
	putFloat64(b[1:], v)
}

func (enc *amf0Encoder) WriteString(v string) {
	copy(enc.initStringHeader(len(v)), v)
}

func (enc *amf0Encoder) WriteBytes(v []byte) {
	copy(enc.initStringHeader(len(v)), v)
}

func (enc *amf0Encoder) WriteTime(v time.Time) {
	b := enc.Next(11)
	b[0] = amf0Date
	putFloat64(b[1:], float64(v.UnixNano()/1e6))
	b[9] = 0
	b[10] = 0
}

func (enc *amf0Encoder) initStringHeader(n int) []byte {
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

func (enc *amf0Encoder) writeString(v string) {
	n := len(v)
	b := enc.Next(n + 2)
	be.PutUint16(b, uint16(n))
	copy(b[2:], v)
}

func (enc *amf0Encoder) writeReference(v reflect.Value) bool {
	// TODO: implement references
	return false
}

func (enc *amf0Encoder) writeSlice(v reflect.Value) (err error) {
	if enc.writeReference(v) {
		return
	}
	b := enc.Next(5)
	b[0] = amf0StrictArray
	n := v.Len()
	be.PutUint32(b[1:], uint32(n))
	for i := 0; i < n; i++ {
		if err = encodeValue(v.Index(i), enc); err != nil {
			return
		}
	}
	return
}

func (enc *amf0Encoder) writeStruct(v reflect.Value) (err error) {
	if enc.writeReference(v) {
		return
	}
	m := getStructMapping(v.Type())
	b := enc.Next(1)
	b[0] = amf0Object
	for _, f := range m.fields {
		r := v.Field(f.index)
		if f.opt && isEmptyValue(r) {
			continue
		}
		enc.writeString(f.name)
		if err = encodeValue(r, enc); err != nil {
			return
		}
	}
	putUint24(enc.Next(3), uint32(amf0ObjectEnd))
	return
}

func (enc *amf0Encoder) writeMap(v reflect.Value) (err error) {
	if enc.writeReference(v) {
		return
	}
	b := enc.Next(1)
	b[0] = amf0Object
	for _, k := range v.MapKeys() {
		switch k.Kind() {
		case reflect.String:
			if n := k.String(); n != "" {
				enc.writeString(n)
				if err = encodeValue(v.MapIndex(k), enc); err != nil {
					return
				}
			}
		default:
			return &errUnsupportedKeyType{k.Type()}
		}
	}
	putUint24(enc.Next(3), uint32(amf0ObjectEnd))
	return
}

type amf0Decoder struct {
	*Reader
	b    []byte
	err  error
	refs map[uint16]interface{}
}

func (dec *amf0Decoder) Decode(v interface{}) error {
	if v == nil {
		return errDecodeNil
	}
	if m, ok := v.(Unmarshaler); ok {
		return m.UnmarshalAMF(dec.Reader)
	}
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Ptr {
		return errDecodeNotPtr
	}
	return decodeValue(r, dec)
}

func (dec *amf0Decoder) Skip() error {
	if dec.next(1) && dec.skipValue(dec.b[0]) {
		return nil
	}
	return dec.err
}

func (dec *amf0Decoder) ReadBool() (v bool, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Boolean:
			v, err = dec.readBool()
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "bool"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadInt() (v int64, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Number:
			var f float64
			f, err = dec.readFloat()
			v = int64(f)
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "int"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadUint() (v uint64, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Number:
			var f float64
			f, err = dec.readFloat()
			v = uint64(f)
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "uint"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadFloat() (v float64, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Number:
			v, err = dec.readFloat()
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "float"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadString() (v string, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0String:
			v, err = dec.readString(false)
		case amf0Null, amf0Undefined:
		case amf0StringExt, amf0Xml:
			v, err = dec.readString(true)
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "string"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadBytes() (v []byte, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0String:
			v, err = dec.readBytes(false)
		case amf0Null, amf0Undefined:
		case amf0StringExt, amf0Xml:
			v, err = dec.readBytes(true)
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "bytes"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) ReadTime() (v time.Time, err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Date:
			v, err = dec.readTime()
		default:
			dec.skipValue(m)
			err = &errUnexpectedMarker{m, "time"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) next(n int) bool {
	dec.b, dec.err = dec.Next(n)
	return dec.err == nil
}

func (dec *amf0Decoder) read() (interface{}, error) {
	if !dec.next(1) {
		return nil, dec.err
	}
	switch m := dec.b[0]; m {
	case amf0Number:
		return dec.readFloat()
	case amf0Boolean:
		return dec.readBool()
	case amf0String:
		return dec.readString(false)
	case amf0Array:
		dec.next(4)
		fallthrough
	case amf0Object:
		return dec.readObject()
	case amf0Null, amf0Undefined:
		return nil, nil
	case amf0Reference:
		return dec.readReference()
	case amf0StrictArray:
		return dec.readStrictArray()
	case amf0Date:
		return dec.readTime()
	case amf0StringExt, amf0Xml:
		return dec.readString(true)
	case amf0Instance:
		if dec.skipString(false) {
			return dec.readObject()
		}
	default:
		dec.err = ErrFormat
	}
	return nil, dec.err
}

func (dec *amf0Decoder) readBool() (v bool, err error) {
	if dec.next(1) {
		v = dec.b[0] != 0
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readFloat() (v float64, err error) {
	if dec.next(8) {
		v = getFloat64(dec.b)
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readString(ext bool) (v string, err error) {
	var b []byte
	if b, err = dec.getBytes(ext); err == nil {
		v = string(b)
	}
	return
}

func (dec *amf0Decoder) readBytes(ext bool) (v []byte, err error) {
	var b []byte
	if b, err = dec.getBytes(ext); err == nil {
		v = make([]byte, len(b))
		copy(v, b)
	}
	return
}

func (dec *amf0Decoder) getBytes(ext bool) (v []byte, err error) {
	if ext {
		if dec.next(4) {
			dec.next(int(be.Uint32(dec.b)))
		}
	} else {
		if dec.next(2) {
			dec.next(int(be.Uint16(dec.b)))
		}
	}
	if dec.err != nil {
		err = dec.err
		return
	}
	v = dec.b
	return
}

func (dec *amf0Decoder) readTime() (v time.Time, err error) {
	if dec.next(10) {
		v = time.Unix(0, int64(getFloat64(dec.b))*1e6).UTC()
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readReference() (v interface{}, err error) {
	// FIXME: implement
	return nil, nil
}

func (dec *amf0Decoder) readStruct(v reflect.Value) (err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Array:
			dec.next(4)
			fallthrough
		case amf0Object:
			err = dec.readStructData(v)
		default:
			err = &errUnexpectedMarker{m, v.Type().String()}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readStructData(v reflect.Value) (err error) {
	m := getStructMapping(v.Type())
	var n string
	for {
		if n, err = dec.readString(false); err != nil {
			return
		} else if n == "" {
			break
		}
		if f := m.names[n]; f != nil {
			decodeValue(v.Field(f.index), dec)
			continue
		}
		if err = dec.Skip(); err != nil {
			return
		}
	}
	dec.next(1)
	return
}

func (dec *amf0Decoder) readMap(v reflect.Value) (err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0Array:
			dec.next(4)
			fallthrough
		case amf0Object:
			err = dec.readMapData(v)
		default:
			err = &errUnexpectedMarker{m, v.Type().String()}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readMapData(v reflect.Value) (err error) {
	e := v.Type().Elem()
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	var n string
	for {
		if n, err = dec.readString(false); err != nil {
			return
		} else if n == "" {
			break
		}
		p := reflect.New(e)
		if err = decodeValue(p, dec); err != nil {
			return
		}
		v.SetMapIndex(reflect.ValueOf(n), p.Elem())
	}
	dec.next(1)
	return
}

func (dec *amf0Decoder) readSlice(v reflect.Value) (err error) {
	if dec.next(1) {
		switch m := dec.b[0]; m {
		case amf0StrictArray:
			err = dec.readSliceData(v)
		default:
			err = &errUnexpectedMarker{m, v.Type().String()}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readSliceData(v reflect.Value) (err error) {
	if !dec.next(4) {
		return dec.err
	}
	k := v.Type().Elem()
	n := int(be.Uint32(dec.b))
	if v.IsNil() {
		v.Set(reflect.MakeSlice(v.Type(), 0, 10))
	}
	for i := 0; i < n; i++ {
		p := reflect.New(k)
		if err = decodeValue(p, dec); err != nil {
			return
		}
		v.Set(reflect.Append(v, p.Elem()))
	}
	return
}

func (dec *amf0Decoder) readStrictArray() (v []interface{}, err error) {
	err = dec.readSliceData(reflect.ValueOf(v))
	return
}

func (dec *amf0Decoder) readObject() (v map[string]interface{}, err error) {
	err = dec.readMapData(reflect.ValueOf(v))
	return
}

func (dec *amf0Decoder) skipValue(m uint8) bool {
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
		return dec.skipStrictArray()
	case amf0Date:
		return dec.next(10)
	case amf0StringExt, amf0Xml:
		return dec.skipString(true)
	case amf0Instance:
		return dec.skipString(false) && dec.skipObject()
	default:
		dec.err = ErrFormat
	}
	return false
}

func (dec *amf0Decoder) skipString(ext bool) bool {
	if ext {
		if dec.next(4) {
			return dec.next(int(be.Uint32(dec.b)))
		}
	} else {
		if dec.next(2) {
			return dec.next(int(be.Uint16(dec.b)))
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

func (dec *amf0Decoder) skipStrictArray() bool {
	if dec.next(4) {
		n := int(be.Uint32(dec.b))
		for n > 0 && dec.Skip() == nil {
			n--
		}
		return n == 0
	}
	return false
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
