package sim

import (
	"math"
	"testing"

	"github.com/McReaper/t7dxbc"
)

// A counting world holds integers in [0, 8) where a world holds floats in
// [-1, 2], in the variables and in the space between them alike; integer and
// bool variables hold the same in both.
func TestCountingWorld(t *testing.T) {
	vec := func(base t7dxbc.VarType) *t7dxbc.Type {
		return &t7dxbc.Type{Class: t7dxbc.ClassVector, Base: base, Rows: 1, Cols: 4}
	}
	ref := &t7dxbc.Reflection{
		Buffers: []t7dxbc.Buffer{{Name: "B", Kind: t7dxbc.CBuffer, Size: 64, Vars: []t7dxbc.Variable{
			{Name: "f", Offset: 0, Size: 16, Type: vec(t7dxbc.TypeFloat)},
			{Name: "u", Offset: 16, Size: 16, Type: vec(t7dxbc.TypeUint)},
			{Name: "b", Offset: 32, Size: 16, Type: vec(t7dxbc.TypeBool)},
		}}},
		Bindings: []t7dxbc.Binding{{Name: "B", Input: t7dxbc.InputCBuffer, Point: 2}},
	}
	for seed := uint32(1); seed <= 20; seed++ {
		plain, counting := NewWorld(ref, seed).CB[2], NewCountingWorld(ref, seed).CB[2]
		if len(plain) != 16 || len(counting) != 16 {
			t.Fatalf("sizes %d, %d", len(plain), len(counting))
		}
		for i := range 16 {
			p, c := plain[i], counting[i]
			switch {
			case i < 4 || i >= 12: // f and the unused last register: floats
				if f := math.Float32frombits(p); f < -1 || f > 2 || p != 0 && p < 8 {
					t.Errorf("seed %d dword %d: %#x isn't a float in [-1, 2]", seed, i, p)
				}
				if c >= 8 {
					t.Errorf("seed %d dword %d: counting %d, want [0, 8)", seed, i, c)
				}
			case i < 8:
				if p >= 8 || c != p {
					t.Errorf("seed %d uint %d: %d and %d", seed, i, p, c)
				}
			default:
				if p > 1 || c != p {
					t.Errorf("seed %d bool %d: %d and %d", seed, i, p, c)
				}
			}
		}
	}
}
