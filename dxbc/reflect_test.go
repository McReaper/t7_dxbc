package dxbc

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/McReaper/t7_dxbc/internal/fixture"
)

// reflectionDiffs compares what the readers find in a shader with the
// reflection header of Microsoft's disassembly of it.
func reflectionDiffs(code []byte, asm string) []string {
	c, err := Parse(code)
	if err != nil {
		return []string{err.Error()}
	}
	ref, err := ReadReflection(c.Chunk("RDEF"))
	if err != nil {
		return []string{"RDEF: " + err.Error()}
	}
	in, out, err := c.Signatures()
	if err != nil {
		return []string{"signature: " + err.Error()}
	}
	h := parseMSHeader(asm)
	var diffs []string
	diff := func(what string, got, want any) {
		if !reflect.DeepEqual(got, want) {
			diffs = append(diffs, fmt.Sprintf("%s:\n  got  %v\n  want %v", what, got, want))
		}
	}
	var vars []msVar
	for _, b := range ref.Buffers {
		for _, v := range b.Vars {
			vars = append(vars, msVar{b.Name, v.Name, v.Offset, v.Size, v.Flags&VariableUsed == 0})
		}
	}
	diff("variables", vars, h.vars)
	var bindings [][]string
	for _, b := range ref.Bindings {
		bindings = append(bindings, []string{b.Name, b.Kind(), b.Format(), b.DimName(), b.Register(), fmt.Sprint(b.Count)})
	}
	diff("bindings", bindings, h.bindings)
	diff("input signature", signatureRows(in, false), trimRows(h.inputs))
	diff("output signature", signatureRows(out, true), trimRows(h.outputs))
	return diffs
}

// msNamedInputs are the registers the disassembler names for inputs
// without a number.
var msNamedInputs = map[SystemValue]string{SVPrimitiveID: "primID"}

// msNamedOutputs are the registers the disassembler names for outputs
// without a number.
var msNamedOutputs = map[SystemValue]string{
	SVDepth: "oDepth", SVCoverage: "oMask", SVDepthGreaterEqual: "oDepthGE", SVDepthLessEqual: "oDepthLE",
	SVStencilRef: "oStencilRef",
}

func signatureRows(es []Element, output bool) [][]string {
	var rows [][]string
	for _, e := range es {
		used := e.RWMask
		if output {
			used = e.Mask &^ e.RWMask // an output records the components it never writes
		}
		letters := func(m uint8) string { return strings.TrimSpace(MaskString(m)) } // the column is trimmed at its ends
		mask, reg, use := letters(e.Mask), fmt.Sprint(e.Register), letters(used)
		if e.Register == NoRegister { // oDepth and the like: the table names the register instead
			mask, reg, use = "N/A", msNamedOutputs[e.SystemValueOf()], "NO"
			if !output {
				reg = msNamedInputs[e.SystemValueOf()]
			}
			if used != 0 {
				use = "YES"
			}
		}
		rows = append(rows, []string{e.Name, fmt.Sprint(e.Index), mask, reg, e.SystemValueOf().String(), e.Type.String(), use})
	}
	return rows
}

func trimRows(rows [][]string) [][]string {
	var out [][]string
	for _, r := range rows {
		row := make([]string, len(r))
		for i, f := range r {
			row[i] = strings.TrimSpace(f)
		}
		out = append(out, row)
	}
	return out
}

func TestReflectionMatchesMicrosoft(t *testing.T) {
	for name, f := range fixture.Shaders(t) {
		for _, d := range reflectionDiffs(f[0], string(f[1])) {
			t.Errorf("%s: %s", name, d)
		}
	}
}

// The structured buffer's element layout, which a decompiler needs to write
// the struct back, is read in full.
func TestStructuredBufferLayout(t *testing.T) {
	c, err := Parse(fixture.Shaders(t)["vs_structured.dxbc"][0])
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReadReflection(c.Chunk("RDEF"))
	if err != nil {
		t.Fatal(err)
	}
	var elem *Buffer
	for i := range ref.Buffers {
		if ref.Buffers[i].Kind == ResourceBindInfo && ref.Buffers[i].Name == "instances" {
			elem = &ref.Buffers[i]
		}
	}
	if elem == nil || len(elem.Vars) != 1 || elem.Vars[0].Name != "$Element" {
		t.Fatalf("no element layout for instances: %+v", ref.Buffers)
	}
	st := elem.Vars[0].Type
	var got []string
	for _, m := range st.Members {
		got = append(got, fmt.Sprintf("%s %s@%d %dx%d", m.Type.HLSLName(), m.Name, m.Offset, m.Type.Rows, m.Type.Cols))
	}
	want := []string{"float4x3 transform@0 4x3", "float4 color@48 1x4", "uint flags@64 1x1"}
	if st.Class != ClassStruct || st.HLSLName() != "Instance" || !reflect.DeepEqual(got, want) {
		t.Errorf("element type %s %v: %v", st.Name, st.Class, got)
	}
	for _, b := range ref.Bindings {
		if b.Name == "instances" && (b.Input != InputStructured || b.NumSamples != 68) {
			t.Errorf("instances binding: %+v (want a structured buffer of stride 68)", b)
		}
	}
}
