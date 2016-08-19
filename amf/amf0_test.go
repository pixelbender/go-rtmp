package amf

import (
	"encoding/hex"
	"math"
	"reflect"
	"testing"
)

type testEncode struct {
	One   int     `amf:"one"`
	Two   string  `amf:"two"`
	Three float64 `amf:"three"`
}

func TestEncodeDecode(t *testing.T) {
	in := &testEncode{
		One:   1,
		Two:   "2",
		Three: 3,
	}
	enc := NewEncoder(0)
	err := enc.Encode(in)
	if err != nil {
		t.Fatal("encode:", err)
	}
	b := enc.Bytes()
	out := &testEncode{}
	dec := NewDecoderBytes(0, b)

	err = dec.Decode(out)
	if err != nil {
		t.Fatal("decode:", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("decode: %v != %v", in, out)
	}
}

func TestPlainTypes(t *testing.T) {
	decode := func(h string, v interface{}) {
		b, err := hex.DecodeString(h)
		if err != nil {
			t.Fatal("hex:", h)
		}
		dec := NewDecoderBytes(0, b)
		r, err := dec.DecodeNext()
		if err != nil {
			t.Fatal("decode:", err, v)
		}
		if r != v {
			t.Fatalf("decode: %v != %v", r, v)
		}
	}
	encode := func(v interface{}, h string) {
		enc := NewEncoder(0)
		err := enc.Encode(v)
		if err != nil {
			t.Fatal("encode:", err, v)
		}
		b := enc.Bytes()
		if r := hex.EncodeToString(b); r != h {
			t.Fatalf("encode: %v %s != %s", v, r, h)
		}
	}
	assert := func(v interface{}, h string, r interface{}) {
		encode(v, h)
		decode(h, r)
	}

	assert("Hello", "02000548656c6c6f", "Hello")
	assert("Привет", "02000cd09fd180d0b8d0b2d0b5d182", "Привет")
	assert(true, "0101", true)
	assert(false, "0100", false)
	assert(uint8(1), "003ff0000000000000", float64(1))
	assert(uint16(2), "004000000000000000", float64(2))
	assert(uint32(3), "004008000000000000", float64(3))
	assert(uint64(4), "004010000000000000", float64(4))
	assert(uint(5), "004014000000000000", float64(5))
	assert(uintptr(6), "004018000000000000", float64(6))
	assert(uint64(math.MaxUint64), "0043f0000000000000", float64(math.MaxUint64))
	assert(0, "000000000000000000", float64(0))
	assert(int8(-1), "00bff0000000000000", float64(-1))
	assert(int16(-2), "00c000000000000000", float64(-2))
	assert(int32(-3), "00c008000000000000", float64(-3))
	assert(int64(-4), "00c010000000000000", float64(-4))
	assert(int(-5), "00c014000000000000", float64(-5))
	assert(math.MaxInt64, "0043e0000000000000", float64(math.MaxInt64))
	assert(float32(1), "003ff0000000000000", float64(1))
	assert(float64(1), "003ff0000000000000", float64(1))
	assert(math.Inf(1), "007ff0000000000000", math.Inf(1))
	assert(math.Inf(-1), "00fff0000000000000", math.Inf(-1))
	assert(math.MaxFloat64, "007fefffffffffffff", float64(math.MaxFloat64))
}
