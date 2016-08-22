package amf

import (
	"errors"
	"reflect"
)

const (
	amf3Undefined    = uint8(0x00) // nil
	amf3Null         = uint8(0x01) // nil
	amf3False        = uint8(0x02) // false
	amf3True         = uint8(0x03) // true
	amf3Integer      = uint8(0x04) // int64
	amf3Double       = uint8(0x05) // float64
	amf3String       = uint8(0x06) // string
	amf3XmlDoc       = uint8(0x07) // string
	amf3Date         = uint8(0x08) // time.Time
	amf3Array        = uint8(0x09) // []interface{} or map[string]interface{}
	amf3Object       = uint8(0x0a) // map[string]interface{}
	amf3Xml          = uint8(0x0b) // string
	amf3ByteArray    = uint8(0x0c) // []byte
	amf3IntVector    = uint8(0x0d) // []int32
	amf3UintVector   = uint8(0x0e) // []uint32
	amf3DoubleVector = uint8(0x0f) // []float64
	amf3ObjectVector = uint8(0x10) // []interface{}
	amf3Dictionary   = uint8(0x11) // map[interface{}]interface{}
)

type amf3Encoder struct {
	writer
}

func (enc *amf3Encoder) Encode(v interface{}) error {
	return enc.EncodeValue(reflect.ValueOf(v))
}

func (enc *amf3Encoder) EncodeValue(v reflect.Value) error {
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
			// TODO: vector types...
			return enc.encodeArray(v)
		}
	case reflect.Struct:
		enc.encodeStruct(v)
	case reflect.Map:
		switch v.Type().Key().Kind() {
		case reflect.String:
			return enc.encodeMap(v)
		default:
			return enc.encodeDict(v)
		}
	case reflect.Ptr:
		return enc.EncodeValue(v.Elem())
	default:
		return errEncUnsType(v.Kind())
	}
	return nil
}

func (enc *amf3Encoder) EncodeNull() {
	enc.Next(1)[0] = amf3Null
}

func (enc *amf3Encoder) EncodeBool(v bool) {
	if b := enc.Next(1); v {
		b[1] = amf3True
	} else {
		b[0] = amf3False
	}
}

func (enc *amf3Encoder) EncodeInt(v int64) {
	if v < 0x40000000 {
		enc.Next(1)[0] = amf3Integer
		putUint29(&enc.writer, uint32(v))
	} else {
		enc.EncodeFloat(float64(v))
	}
}

func (enc *amf3Encoder) EncodeUint(v uint64) {
	if v < 0x40000000 {
		enc.Next(1)[0] = amf3Integer
		putUint29(&enc.writer, uint32(v))
	} else {
		enc.EncodeFloat(float64(v))
	}
}

func (enc *amf3Encoder) EncodeFloat(v float64) {
	b := enc.Next(9)
	b[0] = amf3Double
	putFloat64(b[1:], v)
}

func (enc *amf3Encoder) EncodeString(v string) {
	b := enc.Next(1)
	b[0] = amf3String
	enc.encodeString(v)
}

func (enc *amf3Encoder) EncodeBytes(v []byte) {
	b := enc.Next(1)
	b[0] = amf3ByteArray
	// TODO: reference map
	n := len(v)
	enc.encodeLen(n)
	copy(enc.Next(n), v)
}

func (enc *amf3Encoder) encodeLen(v int) {
	putUint29(&enc.writer, uint32(v)<<1|0x01)
}

func (enc *amf3Encoder) encodeString(v string) {
	// TODO: reference map
	n := len(v)
	enc.encodeLen(n)
	copy(enc.Next(n), v)
}

func (enc *amf3Encoder) encodeEmpty() {
	enc.Next(1)[0] = 0x01
}

func (enc *amf3Encoder) encodeArray(v reflect.Value) (err error) {
	// TODO: reference map
	b := enc.Next(1)
	b[0] = amf3Array
	n := v.Len()
	enc.encodeLen(n)
	enc.encodeEmpty()
	for i := 0; i < n; i++ {
		if err = enc.EncodeValue(v.Index(i)); err != nil {
			return
		}
	}
	return
}

func (enc *amf3Encoder) encodeStruct(v reflect.Value) (err error) {
	// TODO: reference map
	//m := getStructMapping(v.Type())
	//for _, f := range m.fields {
	//
	//}
	return errors.New("not implemented")
}

func (enc *amf3Encoder) encodeMap(v reflect.Value) (err error) {
	// TODO: reference map
	return errors.New("not implemented")
}

func (enc *amf3Encoder) encodeDict(v reflect.Value) (err error) {
	// TODO: reference map
	b := enc.Next(1)
	b[0] = amf3Dictionary
	n := v.Len()
	enc.encodeLen(n)
	enc.Next(1)[0] = 0
	for _, k := range v.MapKeys() {
		if err = enc.EncodeValue(k); err != nil {
			return
		}
		if err = enc.EncodeValue(v.MapIndex(k)); err != nil {
			return
		}
	}
	return
}

type amf3Decoder struct {
	reader
}

func (dec *amf3Decoder) Decode(v interface{}) error {
	return nil
}

func (dec *amf3Decoder) Skip() error {
	return nil
}

func (dec *amf3Decoder) DecodeNext() (interface{}, error) {
	return nil, nil
}

func (dec *amf3Decoder) DecodeInt() (int64, error) {
	return 0, nil
}

func (dec *amf3Decoder) DecodeUint() (uint64, error) {
	return 0, nil
}

func (dec *amf3Decoder) DecodeFloat() (float64, error) {
	return 0, nil
}

func (dec *amf3Decoder) DecodeString() (string, error) {
	return "", nil
}
func (dec *amf3Decoder) DecodeBytes() ([]byte, error) {
	return nil, nil
}

func putUint29(w Writer, v uint32) {
	if v < 0x80 {
		b := w.Next(1)
		b[0] = byte(v)
	} else if v < 0x4000 {
		b := w.Next(2)
		b[0] = byte(v>>7 | 0x80)
		b[1] = byte(v & 0x7f)
	} else if v < 0x200000 {
		b := w.Next(3)
		b[0] = byte(v>>14 | 0x80)
		b[1] = byte(v>>7&0x7f | 0x80)
		b[2] = byte(v & 0x7f)
	} else if v < 0x40000000 {
		b := w.Next(4)
		b[0] = byte(v>>22 | 0x80)
		b[0] = byte(v>>15&0x7f | 0x80)
		b[1] = byte(v>>8&0x7f | 0x80)
		b[2] = byte(v)
	} else {
		// not supported, use double
	}
}
