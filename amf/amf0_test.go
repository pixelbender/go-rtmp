package amf

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

type testMap map[string]interface{}

type testStruct struct {
	Zero  interface{} `amf:"zero"`
	One   int         `amf:"one"`
	Two   string      `amf:"two"`
	Three float64     `amf:"three"`
	Four  int         `amf:"four"`
	Five  []byte      `amf:"five"`
	Six   []int       `amf:"six"`
	Seven time.Time   `amf:"seven"`
	Eight testMap     `amf:"eight"`
	Nine  struct {
		A string `amf:"a"`
	} `amf:"nine"`
}

func TestEncodeDecodeStruct(t *testing.T) {
	ts, _ := time.Parse("02 Jan 06 15:04", "02 Jan 06 15:04")
	four := 4
	five := []byte("five")
	in := &testStruct{
		Zero:  nil,
		One:   1,
		Two:   "2",
		Three: 3,
		Four:  four,
		Five:  five,
		Six:   []int{0, 1, 2, 3, 4, 5},
		Seven: ts,
		Eight: testMap{
			"a": float64(1),
			"b": "2",
			"c": nil,
		},
	}
	in.Nine.A = "inline"
	enc := NewEncoder(0)
	err := enc.Encode(in)
	if err != nil {
		t.Fatal("encode:", err)
	}
	b := enc.Bytes()

	out := &testStruct{}
	dec := NewDecoderBytes(0, b)
	err = dec.Decode(out)
	if err != nil {
		t.Fatal("decode:", err)
	}

	b, _ = json.Marshal(out)
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
		var r interface{}
		err = dec.Decode(&r)
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
	assert(nil, "05", nil)
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

	ts, _ := time.Parse("02 Jan 06 15:04", "02 Jan 06 15:04")
	assert(ts, "0b427088ba56b000000000", ts)
}
