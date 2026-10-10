package hlsl

import (
	"strings"
	"testing"

	"github.com/McReaper/t7_dxbc/dxbc"
)

// sigEl is a signature element: a float, a uint when the name ends in "u",
// a system value when it starts with SV_.
func sigEl(name string, index, reg uint32, mask string) dxbc.Element {
	el := dxbc.Element{Name: name, Index: index, Register: reg, Type: dxbc.ComponentFloat}
	if n, ok := strings.CutSuffix(name, "u"); ok {
		el.Name, el.Type = n, dxbc.ComponentUint
	}
	switch el.Name {
	case "SV_Position":
		el.SystemValue = dxbc.SVPosition
	case "SV_ClipDistance":
		el.SystemValue = dxbc.SVClipDistance
	}
	for _, c := range mask {
		el.Mask |= 1 << strings.IndexRune("xyzw", c)
	}
	return el
}

func layoutString(decls []sigDecl) string {
	lines := make([]string, len(decls))
	for i, d := range decls {
		lines[i] = d.declaration("")
	}
	return strings.Join(lines, "; ")
}

var sigPos = sigEl("SV_Position", 0, 0, "xyzw")

// layoutCases are signatures and the declarations that pack as they say,
// which TestLayoutsCompile has the compiler check (all but the system values,
// which no declaration reproduces).
var layoutCases = []struct {
	name  string
	els   []dxbc.Element
	known map[uint32]string
	want  string
}{
	{"eight float2, each in a register of its own, are an array",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "xy"), sigEl("TEXCOORD", 1, 2, "xy"), sigEl("TEXCOORD", 2, 3, "xy"),
			sigEl("TEXCOORD", 3, 4, "xy"), sigEl("TEXCOORD", 4, 5, "xy"), sigEl("TEXCOORD", 5, 6, "xy"),
			sigEl("TEXCOORD", 6, 7, "xy"), sigEl("TEXCOORD", 7, 8, "xy")}, nil,
		"float4 SV_Position0 : SV_Position0; float2 TEXCOORD0[8] : TEXCOORD0"},
	{"an element after an array shares its first register",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("TEXCOORD", 3, 1, "y"), sigEl("TEXCOORD", 1, 2, "x"),
			sigEl("TEXCOORD", 2, 3, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float TEXCOORD0[3] : TEXCOORD0; float TEXCOORD3 : TEXCOORD3"},
	{"a packing the compiler makes without arrays has none",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "xy"), sigEl("TEXCOORD", 2, 1, "zw"), sigEl("TEXCOORD", 1, 2, "xy")},
		nil, "float4 SV_Position0 : SV_Position0; float2 TEXCOORD0 : TEXCOORD0; float2 TEXCOORD2 : TEXCOORD2; " +
			"float2 TEXCOORD1 : TEXCOORD1"},
	{"no join that moves an element: TEXCOORD6 would go to the array's second register",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("TEXCOORD", 5, 1, "y"), sigEl("TEXCOORD", 6, 1, "z"),
			sigEl("TEXCOORD", 1, 2, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float TEXCOORD0 : TEXCOORD0; float TEXCOORD5 : TEXCOORD5; " +
			"float TEXCOORD6 : TEXCOORD6; float TEXCOORD1[1] : TEXCOORD1"},
	{"integers are an array of their constant group",
		[]dxbc.Element{sigPos, sigEl("TEXCOORDu", 0, 1, "x"), sigEl("TEXCOORDu", 1, 2, "x"), sigEl("TEXCOORD", 5, 3, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; nointerpolation uint TEXCOORD0[2] : TEXCOORD0; float TEXCOORD5 : TEXCOORD5"},
	{"a pixel shader's inputs keep the modes it declares",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "xy"), sigEl("TEXCOORD", 1, 2, "xy"), sigEl("TEXCOORD", 2, 3, "xy"),
			sigEl("TEXCOORD", 3, 4, "x")}, map[uint32]string{2: "", 4: "nointerpolation "},
		"float4 SV_Position0 : SV_Position0; float2 TEXCOORD0[3] : TEXCOORD0; nointerpolation float TEXCOORD3 : TEXCOORD3"},
	{"arrays of different modes stay apart",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("TEXCOORD", 8, 2, "x"), sigEl("TEXCOORD", 1, 3, "x"),
			sigEl("TEXCOORD", 2, 4, "x")}, map[uint32]string{1: "", 2: "centroid ", 3: "", 4: "centroid "},
		"float4 SV_Position0 : SV_Position0; float TEXCOORD0 : TEXCOORD0; centroid float TEXCOORD8 : TEXCOORD8; " +
			"float TEXCOORD1[1] : TEXCOORD1; centroid float TEXCOORD2[1] : TEXCOORD2"},
	{"arrays of different types stay apart",
		[]dxbc.Element{sigPos, sigEl("TEXCOORDu", 0, 1, "x"), sigEl("TEXCOORD", 1, 2, "x"), sigEl("TEXCOORDu", 2, 3, "x")},
		map[uint32]string{2: "nointerpolation "},
		"float4 SV_Position0 : SV_Position0; nointerpolation uint TEXCOORD0 : TEXCOORD0; " +
			"nointerpolation float TEXCOORD1[1] : TEXCOORD1; nointerpolation uint TEXCOORD2[1] : TEXCOORD2"},
	{"an array continues only the same components",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "xy"), sigEl("TEXCOORD", 5, 1, "z"), sigEl("TEXCOORD", 1, 2, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float2 TEXCOORD0 : TEXCOORD0; float TEXCOORD5 : TEXCOORD5; " +
			"float TEXCOORD1[1] : TEXCOORD1"},
	{"an array continues only the same semantic",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("COLOR", 1, 2, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float TEXCOORD0 : TEXCOORD0; float COLOR1[1] : COLOR1"},
	{"an array continues only the next index",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("TEXCOORD", 2, 2, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float TEXCOORD0 : TEXCOORD0; float TEXCOORD2[1] : TEXCOORD2"},
	{"an array continues only in the next register",
		[]dxbc.Element{sigPos, sigEl("TEXCOORD", 0, 1, "x"), sigEl("TEXCOORD", 5, 2, "xyzw"), sigEl("TEXCOORD", 1, 3, "x")},
		nil, "float4 SV_Position0 : SV_Position0; float TEXCOORD0 : TEXCOORD0; float4 TEXCOORD5 : TEXCOORD5; " +
			"float TEXCOORD1[1] : TEXCOORD1"},
	{"a system value is never an array",
		[]dxbc.Element{sigPos, sigEl("SV_ClipDistance", 0, 1, "x"), sigEl("SV_ClipDistance", 1, 2, "x")}, nil,
		"float4 SV_Position0 : SV_Position0; float SV_ClipDistance0 : SV_ClipDistance0; " +
			"float SV_ClipDistance1 : SV_ClipDistance1"},
}

func TestSignatureLayout(t *testing.T) {
	for _, c := range layoutCases {
		if got := layoutString(signatureLayout(c.els, c.known)); got != c.want {
			t.Errorf("%s:\n got  %s\n want %s", c.name, got, c.want)
		}
	}
}

// The layouts that reproduce a packing pack as declared, and only those.
func TestPacksAsDeclared(t *testing.T) {
	a, b := sigEl("TEXCOORD", 0, 0, "x"), sigEl("TEXCOORD", 1, 1, "x")
	if !packsAsDeclared([]sigDecl{{els: []dxbc.Element{a, b}, array: true}}) {
		t.Error("an array of two opens two registers")
	}
	if packsAsDeclared([]sigDecl{{els: []dxbc.Element{a}}, {els: []dxbc.Element{b}}}) {
		t.Error("two scalars share a register")
	}
	if packsAsDeclared([]sigDecl{{els: []dxbc.Element{a}}, {els: []dxbc.Element{sigEl("TEXCOORD", 1, 0, "z")}}}) {
		t.Error("the second scalar goes to .y, not .z")
	}
}

// Elements without a register (SV_Depth) aren't declared as parameters.
func TestPlainLayout(t *testing.T) {
	depth := dxbc.Element{Name: "SV_Depth", Register: dxbc.NoRegister, Mask: 1, SystemValue: dxbc.SVDepth}
	got := layoutString(plainLayout([]dxbc.Element{sigEl("SV_Target", 0, 0, "xyzw"), depth}))
	if got != "float4 SV_Target0 : SV_Target0" {
		t.Errorf("got %s", got)
	}
}

// Each element is named in its array.
func TestFieldNames(t *testing.T) {
	a, b, c := sigEl("TEXCOORD", 0, 1, "xy"), sigEl("TEXCOORD", 1, 2, "xy"), sigEl("COLOR", 0, 3, "x")
	names := fieldNames([]sigDecl{{els: []dxbc.Element{a, b}, array: true}, {els: []dxbc.Element{c}}})
	for s, want := range map[slot]string{{1, 3}: "TEXCOORD0[0]", {2, 3}: "TEXCOORD0[1]", {3, 1}: "COLOR0"} {
		if names[s] != want {
			t.Errorf("%v: %q, want %q", s, names[s], want)
		}
	}
}
