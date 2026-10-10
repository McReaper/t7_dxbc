package hlsl

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/McReaper/t7_dxbc/dxbc"
)

// systemInputs are the inputs a program reads by name, not through a
// signature register: their HLSL type and semantic.
var systemInputs = map[dxbc.OperandType][2]string{
	11: {"uint", "SV_PrimitiveID"}, 32: {"uint3", "SV_DispatchThreadID"}, 33: {"uint3", "SV_GroupID"},
	34: {"uint3", "SV_GroupThreadID"}, 35: {"uint", "SV_Coverage"}, 36: {"uint", "SV_GroupIndex"},
	37: {"uint", "SV_GSInstanceID"},
}

// systemOutputs are the outputs without a register number.
var systemOutputs = map[dxbc.OperandType][2]string{
	12: {"float", "SV_Depth"}, 15: {"uint", "SV_Coverage"}, 38: {"float", "SV_DepthGreaterEqual"},
	39: {"float", "SV_DepthLessEqual"}, 41: {"uint", "SV_StencilRef"},
}

// signatureStructs writes the structs a stage's interface is made of: a
// geometry shader's vertices. Pixel, vertex and compute shaders take their
// parameters directly.
func (e *emitter) signatureStructs() {
	if e.p.Stage == dxbc.StageGeometry {
		e.vertexStructs()
	}
}

// slot is where an element is: its register and components.
type slot struct {
	reg  uint32
	mask uint8
}

// layoutSignatures declares the signatures so that they pack as they were,
// and names each element in them.
func (e *emitter) layoutSignatures() {
	switch e.p.Stage {
	case dxbc.StagePixel:
		e.inDecls, e.outDecls = signatureLayout(e.in, e.inMods), plainLayout(e.out)
	case dxbc.StageVertex:
		e.inDecls, e.outDecls = plainLayout(e.in), signatureLayout(e.out, nil)
	case dxbc.StageGeometry:
		e.inDecls, e.outDecls = signatureLayout(e.in, nil), signatureLayout(e.out, nil)
	}
	e.inFields, e.outFields = fieldNames(e.inDecls), fieldNames(e.outDecls)
}

// fieldNames names each element of declarations: its field, or its array's
// and its index in it.
func fieldNames(decls []sigDecl) map[slot]string {
	names := map[slot]string{}
	for _, d := range decls {
		for i, el := range d.els {
			name := fieldName(d.els[0])
			if d.array {
				name += fmt.Sprintf("[%d]", i)
			}
			names[slot{el.Register, el.Mask}] = name
		}
	}
	return names
}

// declaration declares elements as a parameter or a field: their modifiers,
// type, name and semantic.
func (d sigDecl) declaration(prefix string) string {
	first := d.els[0]
	name := prefix + fieldName(first)
	if d.array {
		name += fmt.Sprintf("[%d]", len(d.els))
	}
	return fmt.Sprintf("%s%s %s : %s%d", d.mode, elementHLSLType(first), name, first.Name, first.Index)
}

func elementHLSLType(el dxbc.Element) string {
	if strings.EqualFold(el.Name, "SV_IsFrontFace") {
		return "bool"
	}
	t := el.Type.String()
	if n := bits.OnesCount8(el.Mask); n > 1 {
		t += fmt.Sprint(n)
	}
	return t
}

// usedOperands lists the operand types the program reads or writes.
func (e *emitter) usedOperands() map[dxbc.OperandType]bool {
	used := map[dxbc.OperandType]bool{}
	var visit func(op *dxbc.Operand)
	visit = func(op *dxbc.Operand) {
		used[op.Type] = true
		for _, i := range op.Index {
			if i.Relative != nil {
				visit(i.Relative)
			}
		}
	}
	for i := range e.p.Instructions {
		for j := range e.p.Instructions[i].Operands {
			visit(&e.p.Instructions[i].Operands[j])
		}
	}
	return used
}

// entry writes the entry point's attributes and parameters.
func (e *emitter) entry() {
	if e.p.Stage == dxbc.StageGeometry {
		e.geometryEntry()
		return
	}
	if e.p.Stage == dxbc.StageCompute {
		e.line("[numthreads(%d, %d, %d)]", e.group[0], e.group[1], e.group[2])
	}
	if e.p.Stage == dxbc.StagePixel && e.global&dxbc.GlobalForceEarlyDepthStencil != 0 {
		e.line("[earlydepthstencil]")
	}
	var params []string
	for _, d := range e.inDecls {
		params = append(params, d.declaration("in_"))
	}
	used := e.usedOperands()
	for _, t := range sortedTypes(systemInputs) {
		if used[t] {
			params = append(params, fmt.Sprintf("%s %s : %s", systemInputs[t][0], t.Prefix(), systemInputs[t][1]))
		}
	}
	for _, d := range e.outDecls {
		params = append(params, "out "+d.declaration("out_"))
	}
	for _, t := range sortedTypes(systemOutputs) {
		if used[t] {
			params = append(params, fmt.Sprintf("out %s %s : %s", systemOutputs[t][0], t.Prefix(),
				e.outputSemantic(systemOutputs[t][1])))
		}
	}
	e.line("void %s(%s)", Entry(e.p.Stage), strings.Join(params, ",\n  "))
	e.line("{")
}

func sortedTypes(m map[dxbc.OperandType][2]string) []dxbc.OperandType {
	out := make([]dxbc.OperandType, 0, len(m))
	for t := range m {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func registerList(prefix string, regs []uint32) string {
	names := make([]string, len(regs))
	for i, r := range regs {
		names[i] = fmt.Sprintf("%s%d", prefix, r)
	}
	return strings.Join(names, ", ")
}

// locals declares the temporaries: each register in the banks it uses, and
// the indexable arrays.
func (e *emitter) locals() {
	var floats, ints []uint32
	for r := range uint32(e.temps) {
		if e.intRegs[r] != 0xF {
			floats = append(floats, r)
		}
		if e.intRegs[r] != 0 {
			ints = append(ints, r)
		}
	}
	if len(floats) > 0 {
		e.line("float4 %s;", registerList("r", floats))
	}
	if len(ints) > 0 {
		e.line("uint4 %s;", registerList("i", ints))
	}
	xs := make([]uint32, 0, len(e.idxTemps))
	for x := range e.idxTemps {
		xs = append(xs, x)
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	for _, x := range xs {
		t := "float4"
		if e.intX[x] {
			t = "uint4"
		}
		e.line("%s x%d[%d];", t, x, e.idxTemps[x][0])
	}
	e.line("")
}

func maskLetters(m uint8) string {
	var sb strings.Builder
	for i := range 4 {
		if m&(1<<i) != 0 {
			sb.WriteByte(components[i])
		}
	}
	return sb.String()
}

// outputSemantic is the semantic of an output without a register as the
// signature spells it (SV_DEPTH, SV_Depth), which the recompiled signature
// keeps.
func (e *emitter) outputSemantic(name string) string {
	for _, el := range e.out {
		if el.Register == dxbc.NoRegister && strings.EqualFold(el.Name, name) {
			return fmt.Sprintf("%s%d", el.Name, el.Index)
		}
	}
	return name
}
