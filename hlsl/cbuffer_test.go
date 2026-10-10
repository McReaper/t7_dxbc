package hlsl

import (
	"testing"

	"github.com/McReaper/t7_dxbc/dxbc"
)

func TestIdentifier(t *testing.T) {
	cases := map[string]string{
		"$Globals": "_Globals", "$Element": "_Element", "tint_2": "tint_2", "Az_az09": "Az_az09", "9lives": "_lives",
		"": "_", "a b": "a_b", "x[3]": "x_3_", "a{b}": "a_b_", "a`b@c": "a_b_c", "é": "_", "a/b:c": "a_b_c",
	}
	for in, want := range cases {
		if got := identifier(in); got != want {
			t.Errorf("identifier(%q) = %q, want %q", in, got, want)
		}
	}
}

var (
	float1Type = &dxbc.Type{Class: dxbc.ClassScalar, Base: dxbc.TypeFloat, Rows: 1, Cols: 1}
	float3Type = &dxbc.Type{Class: dxbc.ClassVector, Base: dxbc.TypeFloat, Rows: 1, Cols: 3}
	uint2Type  = &dxbc.Type{Class: dxbc.ClassVector, Base: dxbc.TypeUint, Rows: 1, Cols: 2}
	// float3x4: three rows of four, as HLSL counts
	rowMajor = &dxbc.Type{Class: dxbc.ClassMatrixRows, Base: dxbc.TypeFloat, Rows: 3, Cols: 4}
	colMajor = &dxbc.Type{Class: dxbc.ClassMatrixColumns, Base: dxbc.TypeFloat, Rows: 3, Cols: 4}
)

// scalarIn finds the component a byte offset reaches in a constant buffer's
// layout (packed: each array element and matrix line on its own register)
// and in a structured buffer's (tight).
func TestScalarIn(t *testing.T) {
	floats := &dxbc.Type{Class: dxbc.ClassScalar, Base: dxbc.TypeFloat, Rows: 1, Cols: 1, Elements: 4}
	item := &dxbc.Type{Class: dxbc.ClassStruct, Name: "Item", Members: []dxbc.Member{
		{Name: "pos", Offset: 0, Type: float3Type},
		{Name: "id", Offset: 12, Type: &dxbc.Type{Class: dxbc.ClassScalar, Base: dxbc.TypeUint, Rows: 1, Cols: 1}},
		{Name: "uv", Offset: 16, Type: uint2Type},
	}}
	items := *item
	items.Elements = 2
	cases := []struct {
		t      *dxbc.Type
		off    uint32
		packed bool
		expr   string
		comp   int // -1: not a vector's
		ok     bool
	}{
		{float1Type, 0, true, "v", -1, true},
		{float1Type, 4, true, "", 0, false},
		{float3Type, 8, true, "v.z", 2, true},
		{float3Type, 12, true, "", 0, false},
		{float3Type, 6, true, "", 0, false},
		{rowMajor, 16, true, "v._m10", -1, true},
		{rowMajor, 12, true, "v._m03", -1, true},
		{rowMajor, 44, true, "v._m23", -1, true},
		{rowMajor, 48, true, "", 0, false},
		{colMajor, 16, true, "v._m01", -1, true},
		{colMajor, 8, true, "v._m20", -1, true},
		{colMajor, 12, true, "", 0, false}, // a column has three rows
		{colMajor, 2, true, "", 0, false},
		{rowMajor, 16, false, "v._m10", -1, true},
		{colMajor, 12, false, "v._m01", -1, true},
		{floats, 16, true, "v[1]", -1, true},
		{floats, 4, true, "", 0, false}, // packed: one element per register
		{floats, 4, false, "v[1]", -1, true},
		{floats, 64, true, "", 0, false},
		{item, 12, true, "v.id", -1, true},
		{item, 20, true, "v.uv.y", 1, true},
		{&items, 24 + 16, false, "v[1].uv.x", 0, true},
		{&items, 32 + 16, true, "v[1].uv.x", 0, true}, // packed: the element pads to 32 bytes
	}
	for _, c := range cases {
		s, ok := scalarIn("v", c.t, c.off, c.packed)
		if ok != c.ok || ok && (s.expr != c.expr || s.comp != c.comp) {
			t.Errorf("%s at %d (packed %v): %q comp %d ok %v, want %q comp %d ok %v", c.t.HLSLName(), c.off, c.packed,
				s.expr, s.comp, ok, c.expr, c.comp, c.ok)
		}
	}
}

func TestTypeSize(t *testing.T) {
	floats := &dxbc.Type{Class: dxbc.ClassScalar, Base: dxbc.TypeFloat, Rows: 1, Cols: 1, Elements: 4}
	cases := []struct {
		t      *dxbc.Type
		packed bool
		want   uint32
	}{
		{float3Type, true, 12},
		{rowMajor, true, 16*2 + 16},
		{colMajor, true, 16*3 + 12},
		{rowMajor, false, 48},
		{floats, true, 16*3 + 4},
		{floats, false, 16},
	}
	for _, c := range cases {
		if got := typeSize(c.t, c.packed); got != c.want {
			t.Errorf("typeSize(%s, packed %v) = %d, want %d", c.t.HLSLName(), c.packed, got, c.want)
		}
	}
}

// A constant buffer's layout finds the variable holding an offset, and
// nothing in a gap or before the first.
func TestCBufferLayoutAt(t *testing.T) {
	l := newCBufferLayout(&dxbc.Buffer{Vars: []dxbc.Variable{
		{Name: "b", Offset: 32, Size: 12, Type: float3Type},
		{Name: "a", Offset: 4, Size: 4, Type: float1Type},
	}})
	cases := []struct {
		off  uint32
		expr string
	}{{0, ""}, {4, "a"}, {8, ""}, {32, "b.x"}, {40, "b.z"}, {44, ""}}
	for _, c := range cases {
		s, ok := l.at(c.off)
		if ok != (c.expr != "") || s.expr != c.expr {
			t.Errorf("at(%d) = %q %v, want %q", c.off, s.expr, ok, c.expr)
		}
	}
}

func TestPackOffset(t *testing.T) {
	for off, want := range map[uint32]string{0: "c0", 4: "c0.y", 24: "c1.z", 44: "c2.w", 48: "c3"} {
		if got := packOffset(off); got != want {
			t.Errorf("packOffset(%d) = %q, want %q", off, got, want)
		}
	}
}
