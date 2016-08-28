package amf

import (
	"errors"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ErrFormat = errors.New("amf: incorrect format")

var timeType = reflect.TypeOf(time.Time{})

var cache map[reflect.Type]*mapping
var mu sync.RWMutex

func getStructMapping(t reflect.Type) (m *mapping) {
	mu.RLock()
	m = cache[t]
	mu.RUnlock()
	if m != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if cache == nil {
		cache = make(map[reflect.Type]*mapping)
	} else if m = cache[t]; m != nil {
		return
	}
	n := t.NumField()
	m = &mapping{
		make(map[string]*field),
		make([]*field, 0, n),
	}
	for i := 0; i < n; i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("amf"), ",")
		name := tag[0]
		if name == "" {
			continue
		}
		fc := &field{index: i, name: name}
		if len(tag) > 1 {
			fc.opt = tag[1] == "omitempty"
		}
		m.names[name] = fc
		m.fields = append(m.fields, fc)
	}
	return
}

type mapping struct {
	names  map[string]*field
	fields []*field
}

type field struct {
	index int
	name  string
	opt   bool
}

func isEmptyValue(v reflect.Value) bool {
	// TODO: detect empty values
	return false
}

type errEncUnsType reflect.Kind

func (err errEncUnsType) Error() string {
	return "amf: encode " + reflect.Kind(err).String()
}

type errUnsupportedType struct {
	t reflect.Type
}

func (err *errUnsupportedType) Error() string {
	return "amf: unsupported type " + err.t.String()
}

type errUnsupportedKeyType struct {
	t reflect.Type
}

func (err *errUnsupportedKeyType) Error() string {
	return "amf: unsupported key type: " + err.t.String()
}

type errUnsupportedMarker struct {
	m uint8
}

func (err *errUnsupportedMarker) Error() string {
	return "amf: unsupported marker: 0x" + strconv.FormatInt(int64(err.m), 16)
}

type errUnsVersion uint8

func (err errUnsVersion) Error() string {
	return "amf: unsupported version: " + strconv.Itoa(int(err))
}

type errUnexpectedMarker struct {
	marker   uint8
	expected string
}

func (err errUnexpectedMarker) Error() string {
	return "amf: marker 0x" + strconv.FormatInt(int64(err.marker), 16) + ", expected 0x" + err.expected
}

var errDecodeNil = errors.New("amf: decoding nil")

var errDecodeNotPtr = errors.New("amf: decoding not a pointer")

type valueEncoder interface {
	Encoder
	writeSlice(v reflect.Value) error
	writeMap(v reflect.Value) error
	writeStruct(v reflect.Value) error
}

func encodeValue(v reflect.Value, enc valueEncoder) error {
	switch v.Kind() {
	case reflect.Invalid:
		enc.WriteNull()
	case reflect.Bool:
		enc.WriteBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		enc.WriteInt(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		enc.WriteUint(v.Uint())
	case reflect.Float32, reflect.Float64:
		enc.WriteFloat(v.Float())
	case reflect.String:
		enc.WriteString(v.String())
	case reflect.Slice, reflect.Array:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			enc.WriteBytes(v.Bytes())
		default:
			return enc.writeSlice(v)
		}
	case reflect.Struct:
		if v.Type() == timeType {
			enc.WriteTime(v.Interface().(time.Time))
		} else {
			return enc.writeStruct(v)
		}
	case reflect.Map:
		return enc.writeMap(v)
	case reflect.Ptr:
		return encodeValue(v.Elem(), enc)
	case reflect.Interface:
		if !v.IsValid() {
			enc.WriteNull()
		} else {
			return encodeValue(v.Elem(), enc)
		}
	default:
		return errEncUnsType(v.Kind())
	}
	return nil
}

type valueDecoder interface {
	Decoder
	readSlice(v reflect.Value) error
	readMap(v reflect.Value) error
	readStruct(v reflect.Value) error
	read() (interface{}, error)
}

func decodeValue(v reflect.Value, dec valueDecoder) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()
	switch v.Kind() {
	case reflect.Bool:
		var r bool
		if r, err = dec.ReadBool(); err == nil {
			v.SetBool(r)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var r int64
		if r, err = dec.ReadInt(); err == nil {
			v.SetInt(r)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var r uint64
		if r, err = dec.ReadUint(); err == nil {
			v.SetUint(r)
		}
	case reflect.Float32, reflect.Float64:
		var r float64
		if r, err = dec.ReadFloat(); err == nil {
			v.SetFloat(r)
		}
	case reflect.String:
		var r string
		if r, err = dec.ReadString(); err == nil {
			v.SetString(r)
		}
	case reflect.Slice:
		switch v.Type().Elem().Kind() {
		case reflect.Uint8:
			var b []byte
			if b, err = dec.ReadBytes(); err == nil {
				v.SetBytes(b)
			}
		default:
			err = dec.readSlice(v)
		}
	case reflect.Struct:
		if v.Type() == timeType {
			var r time.Time
			if r, err = dec.ReadTime(); err == nil {
				v.Set(reflect.ValueOf(r))
			}
		} else {
			err = dec.readStruct(v)
		}
	case reflect.Map:
		switch v.Type().Key().Kind() {
		case reflect.String:
			err = dec.readMap(v)
		default:
			err = &errUnsupportedKeyType{v.Type().Key()}
		}
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		err = decodeValue(v.Elem(), dec)
	case reflect.Interface:
		var r interface{}
		if r, err = dec.read(); r != nil && err == nil {
			v.Set(reflect.ValueOf(r))
		}
	default:
		dec.Skip()
		err = &errUnsupportedType{v.Type()}
	}
	return
}
