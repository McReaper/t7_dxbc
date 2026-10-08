package dxbc

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// systemInputs are the inputs a program reads by name, not through a
// signature register: their HLSL type and semantic.
var systemInputs = map[OperandType][2]string{
	11: {"uint", "SV_PrimitiveID"}, 32: {"uint3", "SV_DispatchThreadID"}, 33: {"uint3", "SV_GroupID"},
	34: {"uint3", "SV_GroupThreadID"}, 35: {"uint", "SV_Coverage"}, 36: {"uint", "SV_GroupIndex"},
	37: {"uint", "SV_GSInstanceID"},
}

// systemOutputs are the outputs without a register number.
var systemOutputs = map[OperandType][2]string{
	12: {"float", "SV_Depth"}, 15: {"uint", "SV_Coverage"}, 38: {"float", "SV_DepthGreaterEqual"},
	39: {"float", "SV_DepthLessEqual"}, 41: {"uint", "SV_StencilRef"},
}

// signatureStructs is a placeholder for stages whose interface is a struct;
// pixel, vertex and compute shaders take their parameters directly.
func (e *emitter) signatureStructs() {}

// paramName is the parameter an element arrives in or leaves through.
func paramName(prefix string, el Element) string {
	return fmt.Sprintf("%s_%s%d", prefix, identifier(el.Name), el.Index)
}

func elementHLSLType(el Element) string {
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
func (e *emitter) usedOperands() map[OperandType]bool {
	used := map[OperandType]bool{}
	var visit func(op *Operand)
	visit = func(op *Operand) {
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
	if e.p.Stage == StageCompute {
		e.line("[numthreads(%d, %d, %d)]", e.group[0], e.group[1], e.group[2])
	}
	var inModes, outModes map[uint32]string
	switch e.p.Stage {
	case StagePixel:
		inModes = packModes(e.in, e.inMods)
	case StageVertex:
		outModes = packModes(e.out, nil)
	}
	var params []string
	for _, el := range e.in {
		if el.Register == NoRegister {
			continue
		}
		params = append(params, fmt.Sprintf("%s%s %s : %s%d", inModes[el.Register], elementHLSLType(el),
			paramName("in", el), el.Name, el.Index))
	}
	used := e.usedOperands()
	for _, t := range sortedTypes(systemInputs) {
		if used[t] {
			params = append(params, fmt.Sprintf("%s %s : %s", systemInputs[t][0], operandPrefixes[t], systemInputs[t][1]))
		}
	}
	for _, el := range e.out {
		if el.Register == NoRegister {
			continue
		}
		params = append(params, fmt.Sprintf("out %s%s %s : %s%d", outModes[el.Register], elementHLSLType(el),
			paramName("out", el), el.Name, el.Index))
	}
	for _, t := range sortedTypes(systemOutputs) {
		if used[t] {
			params = append(params, fmt.Sprintf("out %s %s : %s", systemOutputs[t][0], operandPrefixes[t],
				e.outputSemantic(systemOutputs[t][1])))
		}
	}
	e.line("void main(%s)", strings.Join(params, ",\n  "))
	e.line("{")
}

func sortedTypes(m map[OperandType][2]string) []OperandType {
	out := make([]OperandType, 0, len(m))
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
		if el.Register == NoRegister && strings.EqualFold(el.Name, name) {
			return fmt.Sprintf("%s%d", el.Name, el.Index)
		}
	}
	return name
}
