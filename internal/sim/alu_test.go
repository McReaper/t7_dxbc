package sim

import (
	"math"
	"testing"
)

// The integer and conversion semantics, at the edges the Direct3D 11
// specification defines.
func TestIntegerSemantics(t *testing.T) {
	cases := []struct {
		name      string
		got, want uint32
	}{
		{"ubfe middle", ubfe(8, 4, 0x12345678), 0x67},
		{"ubfe width 0", ubfe(0, 4, 0xFFFFFFFF), 0},
		{"ubfe past the top", ubfe(16, 20, 0xABCDEF01), 0xABC},
		{"ubfe masks width and offset", ubfe(8+32, 4+32, 0x12345678), 0x67},
		{"ibfe sign-extends", ibfe(4, 0, 0xF), 0xFFFFFFFF},
		{"ibfe positive", ibfe(4, 4, 0x70), 7},
		{"ibfe past the top", ibfe(8, 28, 0x80000000), 0xFFFFFFF8},
		{"bfi", bfiBits(8, 8, 0xAB, 0x12345678), 0x1234AB78},
		{"bfi width 0 keeps base", bfiBits(0, 8, 0xAB, 0x12345678), 0x12345678},
		{"firstbit_hi of 1 counts from the top", firstbitHi(1), 31},
		{"firstbit_hi of the top bit", firstbitHi(0x80000000), 0},
		{"firstbit_hi of 0", firstbitHi(0), 0xFFFFFFFF},
		{"firstbit_lo", firstbitLo(8), 3},
		{"firstbit_lo of 0", firstbitLo(0), 0xFFFFFFFF},
		{"firstbit_shi of -1", firstbitShi(0xFFFFFFFF), 0xFFFFFFFF},
		{"firstbit_shi of -2", firstbitShi(0xFFFFFFFE), 31},
		{"firstbit_shi of 1", firstbitShi(1), 31},
		{"ftoi truncates", ftoi(-1.5), 0xFFFFFFFF},
		{"ftoi NaN", ftoi(float32(math.NaN())), 0},
		{"ftoi saturates up", ftoi(3e9), 0x7FFFFFFF},
		{"ftoi saturates down", ftoi(-3e9), 0x80000000},
		{"ftou negative", ftou(-1), 0},
		{"ftou saturates", ftou(5e9), 0xFFFFFFFF},
		{"ftou truncates", ftou(2.9), 2},
		{"saturate NaN", saturate(math.Float32bits(float32(math.NaN()))), 0},
		{"saturate above", saturate(math.Float32bits(2)), math.Float32bits(1)},
		{"saturate below", saturate(math.Float32bits(-1)), 0},
		{"flush keeps the sign", flush(0x80000001), 0x80000000},
		{"flush keeps a normal", flush(0x00800000), 0x00800000},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %#x, want %#x", c.name, c.got, c.want)
		}
	}
}

func TestFloatSemantics(t *testing.T) {
	nan := float32(math.NaN())
	if fmin(nan, 2) != 2 || fmin(2, nan) != 2 || fmax(nan, 3) != 3 || fmax(3, nan) != 3 {
		t.Error("min and max must return the number when the other is NaN")
	}
	if fmin(-1, 2) != -1 || fmax(-1, 2) != 2 {
		t.Error("min and max of numbers")
	}
}

// Half floats: every one converts to a float and back to itself, and floats
// round to the nearest half, ties to even.
func TestHalfFloats(t *testing.T) {
	for h := range uint32(1 << 16) {
		f := halfToFloat(uint16(h))
		if f != f {
			continue // NaN payloads are not kept
		}
		if back := floatToHalf(f); back != uint16(h) {
			t.Fatalf("%#04x -> %g -> %#04x", h, f, back)
		}
	}
	cases := []struct {
		f    float32
		want uint16
	}{
		{1, 0x3C00}, {-2, 0xC000}, {65504, 0x7BFF}, {65520, 0x7C00}, {1e9, 0x7C00},
		{5.9604645e-8, 0x0001}, {2.9e-8, 0}, {float32(math.NaN()), 0x7E00},
		{1 + 1.0/2048, 0x3C00}, {1 + 3.0/2048, 0x3C02},
	}
	for _, c := range cases {
		if got := floatToHalf(c.f); got != c.want {
			t.Errorf("floatToHalf(%g) = %#04x, want %#04x", c.f, got, c.want)
		}
	}
}
