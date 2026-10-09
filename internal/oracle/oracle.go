//go:build windows

// Package oracle recompiles a decompiled shader with Microsoft's compiler
// and compares what an engine binding the two would see: their constant
// buffers and the variables they read, their resources, their signatures,
// the declarations whose controls change what a program does, and the kinds
// of barrier it waits at. What the programs compute is internal/sim's to
// compare.
package oracle

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/McReaper/dxbc"
	"github.com/McReaper/dxbc/internal/d3dc"
)

// Available reports whether the compiler can be loaded.
func Available() bool { return d3dc.Available() }

var targets = map[dxbc.Stage]string{dxbc.StagePixel: "ps", dxbc.StageVertex: "vs", dxbc.StageCompute: "cs",
	dxbc.StageGeometry: "gs"}

// Recompile decompiles a shader and compiles the HLSL again for the same
// target. Its error says which step failed: "decompile: …" or "compile: …".
func Recompile(code []byte) (src string, out []byte, err error) {
	src, err = dxbc.Decompile(code)
	if err != nil {
		return src, nil, fmt.Errorf("decompile: %w", err)
	}
	c, err := dxbc.Parse(code)
	if err != nil {
		return src, nil, fmt.Errorf("decompile: %w", err)
	}
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return src, nil, fmt.Errorf("decompile: %w", err)
	}
	target := fmt.Sprintf("%s_%d_%d", targets[p.Stage], p.Major, p.Minor)
	out, err = d3dc.Compile([]byte(src), "decompiled.hlsl", "main", target)
	if err != nil {
		return src, nil, fmt.Errorf("compile: %w", err)
	}
	return src, out, nil
}

// Interface is what a shader exposes, a line per item, in a form two
// shaders compare by.
type Interface struct {
	Buffers, Bindings, Inputs, Outputs []string
	// Declarations are those whose controls change what the program does:
	// a UAV's coherence, a resource's shape, a sampler's mode, an input's
	// interpolation, the global flags (an early depth test…)
	Declarations []string
	// Barriers are the kinds of sync the program has: what memory it waits
	// for decides what other threads see, which a sequential simulation
	// can't tell
	Barriers []string
}

// InterfaceOf reads a compiled shader's interface.
func InterfaceOf(code []byte) (Interface, error) {
	var s Interface
	c, err := dxbc.Parse(code)
	if err != nil {
		return s, err
	}
	ref := &dxbc.Reflection{}
	if rdef := c.Chunk("RDEF"); rdef != nil {
		if ref, err = dxbc.ReadReflection(rdef); err != nil {
			return s, err
		}
	}
	for _, b := range ref.Buffers {
		s.Buffers = append(s.Buffers, fmt.Sprintf("%s kind=%d size=%d", b.Name, b.Kind, b.Size))
		for _, v := range b.Vars {
			s.Buffers = append(s.Buffers, fmt.Sprintf("  %s %s @%d+%d used=%v", v.Type.HLSLName(), v.Name, v.Offset,
				v.Size, v.Flags&dxbc.VariableUsed != 0))
		}
	}
	for _, b := range ref.Bindings {
		s.Bindings = append(s.Bindings, fmt.Sprintf("%s %s %s %s %s count=%d", b.Name, b.Kind(), b.Format(), b.DimName(),
			b.Register(), b.Count))
	}
	in, out, err := c.Signatures()
	if err != nil {
		return s, err
	}
	s.Inputs, s.Outputs = signature(in), signature(out)
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return s, err
	}
	s.Declarations, s.Barriers = programDeclarations(p)
	return s, nil
}

func signature(els []dxbc.Element) []string {
	var rows []string
	for _, el := range els {
		rows = append(rows, fmt.Sprintf("%s%d %v %v r%d %s", el.Name, el.Index, el.SystemValue, el.Type, el.Register,
			dxbc.MaskString(el.Mask)))
	}
	return rows
}

// programDeclarations lists the declarations that carry meaning beyond the
// reflection, as the disassembler writes them, and the kinds of barrier. An
// input declaration keeps only its register and interpolation mode, as the
// components it lists follow what the code reads.
func programDeclarations(p *dxbc.Program) (decls, barriers []string) {
	seen := map[string]bool{}
	for i := range p.Instructions {
		in := &p.Instructions[i]
		switch in.Op {
		case dxbc.OpDclUnorderedAccessViewTyped, dxbc.OpDclUnorderedAccessViewRaw, dxbc.OpDclUnorderedAccessViewStructured,
			dxbc.OpDclResource, dxbc.OpDclResourceRaw, dxbc.OpDclResourceStructured, dxbc.OpDclSampler,
			dxbc.OpDclGlobalFlags:
			decls = append(decls, in.Disassemble())
		case dxbc.OpDclInputPs, dxbc.OpDclInputPsSiv, dxbc.OpDclInputPsSgv:
			if len(in.Operands) > 0 && len(in.Operands[0].Index) > 0 {
				decls = append(decls, fmt.Sprintf("v%d mode %d", in.Operands[0].Index[0].Imm, in.InterpolationMode()))
			}
		case dxbc.OpSync:
			if m := in.Disassemble(); !seen[m] {
				seen[m] = true
				barriers = append(barriers, m)
			}
		}
	}
	sort.Strings(decls)
	sort.Strings(barriers)
	return decls, barriers
}

// Diffs compares two shaders' interfaces: a line per part that differs,
// starting with the part's name and how it first differs.
func Diffs(want, got []byte) []string {
	a, err := InterfaceOf(want)
	if err != nil {
		return []string{"original: " + err.Error()}
	}
	b, err := InterfaceOf(got)
	if err != nil {
		return []string{"recompiled: " + err.Error()}
	}
	var diffs []string
	for _, f := range []struct {
		name   string
		wa, ga []string
	}{
		{"buffers", a.Buffers, b.Buffers}, {"bindings", a.Bindings, b.Bindings}, {"inputs", a.Inputs, b.Inputs},
		{"outputs", a.Outputs, b.Outputs}, {"declarations", a.Declarations, b.Declarations},
		{"barriers", a.Barriers, b.Barriers},
	} {
		if !reflect.DeepEqual(f.wa, f.ga) {
			diffs = append(diffs, fmt.Sprintf("%s: %s\n  want %s\n  got  %s", f.name, firstDifference(f.wa, f.ga),
				strings.Join(f.wa, " | "), strings.Join(f.ga, " | ")))
		}
	}
	return diffs
}

// firstDifference names how two lists first differ: a variable whose use
// changed, a line missing or out of order.
func firstDifference(want, got []string) string {
	for i := range want {
		if i >= len(got) {
			return "missing " + strings.Fields(want[i])[0]
		}
		if want[i] == got[i] {
			continue
		}
		w, g := strings.Fields(want[i]), strings.Fields(got[i])
		if len(w) == len(g) && len(w) > 1 && strings.HasPrefix(w[len(w)-1], "used=") &&
			strings.Join(w[:len(w)-1], " ") == strings.Join(g[:len(g)-1], " ") {
			return "use changed (" + w[len(w)-1] + ")"
		}
		return "differs: " + want[i] + " vs " + got[i]
	}
	return "extra " + got[len(want)]
}
