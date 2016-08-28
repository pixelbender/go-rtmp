package amf

//
//import (
//	"reflect"
//	"time"
//)
//
//const (
//	amf3Undefined    = uint8(0x00) // nil
//	amf3Null         = uint8(0x01) // nil
//	amf3False        = uint8(0x02) // false
//	amf3True         = uint8(0x03) // true
//	amf3Integer      = uint8(0x04) // int64
//	amf3Double       = uint8(0x05) // float64
//	amf3String       = uint8(0x06) // string
//	amf3XmlDoc       = uint8(0x07) // string
//	amf3Date         = uint8(0x08) // time.Time
//	amf3Array        = uint8(0x09) // []interface{} or map[string]interface{}
//	amf3Object       = uint8(0x0a) // map[string]interface{}
//	amf3Xml          = uint8(0x0b) // string
//	amf3ByteArray    = uint8(0x0c) // []byte
//	amf3IntVector    = uint8(0x0d) // []int32
//	amf3UintVector   = uint8(0x0e) // []uint32
//	amf3DoubleVector = uint8(0x0f) // []float64
//	amf3ObjectVector = uint8(0x10) // []interface{}
//	amf3Dictionary   = uint8(0x11) // map[interface{}]interface{}
//)
//
//type amf3Encoder struct {
//	*Writer
//}
//
//func (enc *amf3Encoder) Encode(v interface{}) error {
//	return encodeValue(reflect.ValueOf(v), enc)
//}
//
//func (enc *amf3Encoder) WriteNull() {
//	enc.Next(1)[0] = amf3Null
//}
//
//func (enc *amf3Encoder) WriteBool(v bool) {
//	if b := enc.Next(1); v {
//		b[1] = amf3True
//	} else {
//		b[0] = amf3False
//	}
//}
//
//func (enc *amf3Encoder) WriteInt(v int64) {
//	if 0 <= v && v < 0x40000000 {
//		enc.Next(1)[0] = amf3Integer
//		enc.writeUint29(uint32(v))
//	} else {
//		enc.WriteFloat(float64(v))
//	}
//}
//
//func (enc *amf3Encoder) WriteUint(v uint64) {
//	if v < 0x40000000 {
//		enc.Next(1)[0] = amf3Integer
//		enc.writeUint29(uint32(v))
//	} else {
//		enc.WriteFloat(float64(v))
//	}
//}
//
//func (enc *amf3Encoder) WriteFloat(v float64) {
//	b := enc.Next(9)
//	b[0] = amf3Double
//	putFloat64(b[1:], v)
//}
//
//func (enc *amf3Encoder) WriteString(v string) {
//	b := enc.Next(1)
//	b[0] = amf3String
//	enc.writeString(v)
//}
//
//func (enc *amf3Encoder) WriteBytes(v []byte) {
//	b := enc.Next(1)
//	b[0] = amf3ByteArray
//	if enc.writeReference(v) {
//		return
//	}
//	n := len(v)
//	enc.writeLen(n)
//	copy(enc.Next(n), v)
//}
//
//func (enc *amf3Encoder) WriteTime(v time.Time) {
//	b := enc.Next(1)
//	b[0] = amf0Date
//	if enc.writeReference(v) {
//		return
//	}
//	b = enc.Next(9)
//	b[0] = 0x01
//	putFloat64(b[1:], float64(v.UnixNano()/1e6))
//}
//
//func (enc *amf3Encoder) writeReference(v interface{}) bool {
//	// TODO: implement references
//	return false
//}
//
//func (enc *amf3Encoder) writeUint29(v uint32) {
//	if v < 0x80 {
//		b := enc.Next(1)
//		b[0] = byte(v)
//	} else if v < 0x4000 {
//		b := enc.Next(2)
//		b[0] = byte(v>>7 | 0x80)
//		b[1] = byte(v & 0x7f)
//	} else if v < 0x200000 {
//		b := enc.Next(3)
//		b[0] = byte(v>>14 | 0x80)
//		b[1] = byte(v>>7&0x7f | 0x80)
//		b[2] = byte(v & 0x7f)
//	} else if v < 0x40000000 {
//		b := enc.Next(4)
//		b[0] = byte(v>>22 | 0x80)
//		b[0] = byte(v>>15&0x7f | 0x80)
//		b[1] = byte(v>>8&0x7f | 0x80)
//		b[2] = byte(v)
//	} else {
//		// Not supported, use float
//	}
//}
//
//func (enc *amf3Encoder) writeLen(v int) {
//	enc.writeUint29(uint32(v)<<1|0x01)
//}
//
//func (enc *amf3Encoder) writeString(v string) {
//	if enc.writeReference(v) {
//		return
//	}
//	n := len(v)
//	enc.writeLen(n)
//	copy(enc.Next(n), v)
//}
//
//func (enc *amf3Encoder) encodeEmpty() {
//	enc.Next(1)[0] = 0x01
//}
//
//func (enc *amf3Encoder) encodeSlice(v reflect.Value) (err error) {
//	b := enc.Next(1)
//	b[0] = amf3Array
//	if enc.writeReference(v) {
//		return
//	}
//	n := v.Len()
//	enc.writeLen(n)
//	enc.encodeEmpty()
//	for i := 0; i < n; i++ {
//		if err = encodeValue(v.Index(i), enc); err != nil {
//			return
//		}
//	}
//	return
//}
//
//func (enc *amf3Encoder) encodeStruct(v reflect.Value) (err error) {
//	b := enc.Next(1)
//	b[0] = amf3Array
//	if enc.writeReference(v) {
//		return
//	}
//	// .....0 = ref
//	// ....01 = traits-ref ()
//	// ..d011 = traits (number of sealed traits member names)
//	// ...111 = traits-ext
//
//	//
//	// TODO: reference map
//	//m := getStructMapping(v.Type())
//	//for _, f := range m.fields {
//	//
//	//}
//	return errors.New("not implemented")
//}
//
//func (enc *amf3Encoder) encodeMap(v reflect.Value) (err error) {
//	// TODO: reference map
//	return errors.New("not implemented")
//}
//
//func (enc *amf3Encoder) encodeDict(v reflect.Value) (err error) {
//	// TODO: reference map
//	b := enc.Next(1)
//	b[0] = amf3Dictionary
//	n := v.Len()
//	enc.writeLen(n)
//	enc.Next(1)[0] = 0
//	for _, k := range v.MapKeys() {
//		if err = enc.EncodeValue(k); err != nil {
//			return
//		}
//		if err = enc.EncodeValue(v.MapIndex(k)); err != nil {
//			return
//		}
//	}
//	return
//}
//
////
////type amf3Decoder struct {
////	*Reader
////}
////
////func (dec *amf3Decoder) Decode(v interface{}) error {
////	return nil
////}
////
////func (dec *amf3Decoder) Skip() error {
////	return nil
////}
////
////func (dec *amf3Decoder) DecodeNext() (interface{}, error) {
////	return nil, nil
////}
////
////func (dec *amf3Decoder) DecodeInt() (int64, error) {
////	return 0, nil
////}
////
////func (dec *amf3Decoder) DecodeUint() (uint64, error) {
////	return 0, nil
////}
////
////func (dec *amf3Decoder) DecodeFloat() (float64, error) {
////	return 0, nil
////}
////
////func (dec *amf3Decoder) DecodeString() (string, error) {
////	return "", nil
////}
////func (dec *amf3Decoder) DecodeBytes() ([]byte, error) {
////	return nil, nil
////}
////
