package amf

import (
	"encoding/binary"
	"log"
	"math"
	"reflect"
	"runtime"
	"runtime/debug"
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
	case reflect.Slice:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			enc.EncodeBytes(v.Bytes())
		default:
			return enc.encodeSlice(v)
		}
	case reflect.Struct:
		if v.Type() == timeType {
			enc.EncodeTime(v.Interface().(time.Time))
		} else {
			return enc.encodeStruct(v)
		}
	case reflect.Map:
		return enc.encodeMap(v)
	case reflect.Ptr:
		return enc.encodeValue(v.Elem())
	case reflect.Interface:
		if !v.IsValid() {
			enc.EncodeNull()
		} else {
			return enc.encodeValue(v.Elem())
		}
	default:
		return errEncUnsType(v.Kind())
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

func (enc *amf0Encoder) writeString(v string) {
	n := len(v)
	b := enc.Next(n + 2)
	be.PutUint16(b, uint16(n))
	copy(b[2:], v)
}

func (enc *amf0Encoder) EncodeTime(t time.Time) {
	b := enc.Next(11)
	b[0] = amf0Date
	putFloat64(b[1:], float64(t.UnixNano()/1e6))
	b[9] = 0
	b[10] = 0
}

func (enc *amf0Encoder) encodeSlice(v reflect.Value) (err error) {
	// TODO: reference map
	b := enc.Next(5)
	b[0] = amf0StrictArray
	n := v.Len()
	be.PutUint32(b[1:], uint32(n))
	for i := 0; i < n; i++ {
		if err = enc.encodeValue(v.Index(i)); err != nil {
			return
		}
	}
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
		enc.writeString(f.name)
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
				enc.writeString(s)
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
	reader
	b   []byte
	err error
}

func (dec *amf0Decoder) next(n int) bool {
	dec.b, dec.err = dec.Peek(n)
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
	return dec.decodeValue(r)
}

func (dec *amf0Decoder) decodeValue(v reflect.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("%v", r)
			debug.PrintStack()
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	switch v.Kind() {
	case reflect.Bool:
		var r bool
		if r, err = dec.DecodeBool(); err != nil {
			return
		}
		v.SetBool(r)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var r int64
		if r, err = dec.DecodeInt(); err != nil {
			return
		}
		v.SetInt(r)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var r uint64
		if r, err = dec.DecodeUint(); err != nil {
			return
		}
		v.SetUint(r)

	case reflect.Float32, reflect.Float64:
		var r float64
		if r, err = dec.DecodeFloat(); err != nil {
			return
		}
		v.SetFloat(r)
	case reflect.String:
		var r string
		if r, err = dec.DecodeString(); err != nil {
			return
		}
		v.SetString(r)
	//case reflect.Array:
	// TODO: array support
	case reflect.Slice:
		k := v.Type().Elem().Kind()
		switch k {
		case reflect.Uint8:
			var b []byte
			if b, err = dec.DecodeBytes(); err != nil {
				return
			}
			v.SetBytes(b)
		default:
			err = dec.decodeSlice(v)
		}
	case reflect.Struct:
		if v.Type() == timeType {
			var r time.Time
			if r, err = dec.DecodeTime(); err != nil {
				return
			}
			v.Set(reflect.ValueOf(r))
		} else {
			err = dec.decodeStruct(v)
		}
	case reflect.Map:
		k := v.Type().Key().Kind()
		switch k {
		case reflect.String:
			err = dec.decodeMap(v)
		default:
			err = errUnsKeyType(k)
		}
	case reflect.Ptr:
		e := v.Type().Elem()
		if v.IsNil() {
			v.Set(reflect.New(e))
		}
		if err = dec.decodeValue(v.Elem()); err != nil {
			return
		}
	case reflect.Interface:
		var r interface{}
		if r, err = dec.read(); err != nil {
			return
		}
		if r != nil {
			v.Set(reflect.ValueOf(r))
		}
	default:
		dec.Skip()
		err = errDecUnsType(v.Kind())
	}
	return
}

func (dec *amf0Decoder) DecodeBool() (v bool, err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0Boolean:
			v, err = dec.readBool()
		default:
			dec.skipMarker(m)
			err = &errDecMarkerKind{m, "bool"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) DecodeInt() (v int64, err error) {
	var f float64
	if f, err = dec.DecodeFloat(); err == nil {
		v = int64(f)
	}
	return
}

func (dec *amf0Decoder) DecodeUint() (v uint64, err error) {
	var f float64
	if f, err = dec.DecodeFloat(); err == nil {
		v = uint64(f)
	}
	return
}

func (dec *amf0Decoder) DecodeFloat() (v float64, err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0Number:
			v, err = dec.readFloat()
		default:
			err = &errDecMarkerKind{m, "number"}
			dec.skipMarker(m)
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) DecodeString() (v string, err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0String:
			v, err = dec.readString(false)
		case amf0Null, amf0Undefined:
		case amf0StringExt, amf0Xml:
			v, err = dec.readString(true)
		default:
			err = &errDecMarkerKind{m, "string"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) DecodeBytes() (v []byte, err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0String:
			v, err = dec.readBytes(false, true)
		case amf0Null, amf0Undefined:
		case amf0StringExt, amf0Xml:
			v, err = dec.readBytes(true, true)
		default:
			err = &errDecMarkerKind{m, "string"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) DecodeTime() (v time.Time, err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0Date:
			v, err = dec.readTime()
		default:
			err = &errDecMarkerKind{m, "time"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) read() (interface{}, error) {
	if !dec.next(1) {
		return nil, dec.err
	}
	m := dec.b[0]
	switch m {
	case amf0Number:
		return dec.readFloat()
	case amf0Boolean:
		return dec.readBool()
	case amf0String:
		return dec.readString(false)
	case amf0Object:
		return dec.readMap()
	case amf0Null, amf0Undefined:
		return nil, nil
	//case amf0Reference:
	// TODO: add support for refs
	case amf0Array:
		dec.next(4)
		return dec.readMap()
	case amf0StrictArray:
		return dec.readSlice()
	case amf0Date:
		return dec.readTime()
	case amf0StringExt, amf0Xml:
		return dec.readString(true)
	case amf0Instance:
		dec.skipString(false)
		return dec.readMap()
	default:
		dec.err = errUnsMarker(m)
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
	b, err = dec.readBytes(ext, false)
	v = string(b)
	return
}

func (dec *amf0Decoder) readBytes(ext bool, clone bool) (v []byte, err error) {
	if ext {
		if dec.next(4) {
			if n := int(be.Uint32(dec.b)); dec.next(n) {
				v = dec.b
			}
		}
	} else {
		if dec.next(2) {
			if n := int(be.Uint16(dec.b)); dec.next(n) {
				v = dec.b
			}
		}
	}
	if dec.err != nil {
		err = dec.err
	} else if clone {
		r := make([]byte, len(v))
		copy(r, v)
		v = r
	}
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

func (dec *amf0Decoder) decodeStruct(v reflect.Value) (err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0Array:
			dec.next(4)
			fallthrough
		case amf0Object:
			err = dec.readStructData(v)
		default:
			err = &errDecMarkerKind{m, "struct"}
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
			dec.decodeValue(v.Field(f.index))
			continue
		}
		if err = dec.Skip(); err != nil {
			return
		}
	}
	dec.next(1)
	return
}

func (dec *amf0Decoder) decodeMap(v reflect.Value) (err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0Array:
			dec.next(4)
			fallthrough
		case amf0Object:
			err = dec.readMapData(v)
		default:
			err = &errDecMarkerKind{m, "struct"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readMap() (v map[string]interface{}, err error) {
	v = make(map[string]interface{})
	if err = dec.readMapData(reflect.ValueOf(v)); err != nil {
		return
	}
	return
}

func (dec *amf0Decoder) readMapData(v reflect.Value) (err error) {
	e := v.Type().Elem()
	if v.IsNil() {
		m := reflect.MakeMap(v.Type())
		v.Set(m)
	}
	var n string
	for {
		if n, err = dec.readString(false); err != nil {
			return
		} else if n == "" {
			break
		}
		p := reflect.New(e)
		if err = dec.decodeValue(p); err != nil {
			return
		}
		v.SetMapIndex(reflect.ValueOf(n), p.Elem())
	}
	dec.next(1)
	return
}

func (dec *amf0Decoder) decodeSlice(v reflect.Value) (err error) {
	if dec.next(1) {
		m := dec.b[0]
		switch m {
		case amf0StrictArray:
			err = dec.readSliceData(v)
		default:
			err = &errDecMarkerKind{m, "slice"}
		}
	} else {
		err = dec.err
	}
	return
}

func (dec *amf0Decoder) readSlice() (v []interface{}, err error) {
	if err = dec.readSliceData(reflect.ValueOf(v)); err != nil {
		return
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
		cap := n
		if cap > 1024 {
			cap = 1024
		}
		v.Set(reflect.MakeSlice(v.Type(), 0, cap))
	}
	for i := 0; i < n; i++ {
		p := reflect.New(k)
		if err = dec.decodeValue(p); err != nil {
			return
		}
		v.Set(reflect.Append(v, p.Elem()))
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
