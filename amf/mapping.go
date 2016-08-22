package amf

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

type errDecUnsType reflect.Kind

func (err errDecUnsType) Error() string {
	return "amf: decode " + reflect.Kind(err).String()
}

type errUnsKeyType reflect.Kind

func (err errUnsKeyType) Error() string {
	return "amf: unsupported map key type: " + reflect.Kind(err).String()
}

type errUnsMarker uint8

func (err errUnsMarker) Error() string {
	return "amf: unsupported marker: 0x" + strconv.FormatInt(int64(err), 16)
}

type errUnsVersion uint8

func (err errUnsVersion) Error() string {
	return "amf: unsupported version: " + strconv.Itoa(int(err))
}

type errDecMarkerKind struct {
	m    uint8
	kind string
}

func (err errDecMarkerKind) Error() string {
	return "amf: marker 0x" + strconv.FormatInt(int64(err.m), 16) + " is not a " + err.kind
}

var errDecodeNil = errors.New("amf: Decode(nil)")
var errDecodeNotPtr = errors.New("amf: Decode(not a pointer)")
