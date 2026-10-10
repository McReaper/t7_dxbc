package t7dxbc

import (
	"fmt"
	"strings"
)

// Disassemble writes the program's instructions the way Microsoft's
// disassembler does, one per line, without its indentation: the decoder is
// checked against that output on every shader of the oracle corpus.
func (p *Program) Disassemble() []string {
	out := make([]string, 0, len(p.Instructions))
	for i := range p.Instructions {
		out = append(out, p.Instructions[i].Disassemble())
	}
	return out
}

// Disassemble writes one instruction.
func (in *Instruction) Disassemble() string {
	if in.Op == OpCustomdata {
		return disasmCustomdata(in)
	}
	if f, ok := declFormats[in.Op]; ok {
		return f(in)
	}
	name := mnemonic(in)
	if len(in.Operands) == 0 {
		return name
	}
	style := styleOf(in.Op)
	ops := make([]string, len(in.Operands))
	for i := range in.Operands {
		ops[i] = formatOperand(&in.Operands[i], operandStyle(in.Op, i, style))
	}
	return name + " " + strings.Join(ops, ", ")
}

// mnemonic is the opcode's name with its controls: _sat, the _z/_nz of a
// conditional, the resource of an _indexable sample or load.
func mnemonic(in *Instruction) string {
	name := opInfos[in.Op].name
	if n, ok := mnemonicOverrides[in.Op]; ok {
		name = n
	}
	if conditional[in.Op] {
		if in.TestsNonZero() {
			name += "_nz"
		} else {
			name += "_z"
		}
	}
	if in.Saturate() {
		name += "_sat"
	}
	if in.Op == OpSync {
		name += syncFlags(in.SyncFlags())
	}
	name += extendedSuffix(in)
	switch {
	case in.Op == OpResinfo:
		name += [...]string{"", "_rcpFloat", "_uint", ""}[in.ResinfoReturn()]
	case in.Op == OpSampleInfo && in.SampleinfoUint():
		name += "_uint"
	}
	return name
}

// syncFlags writes what a sync waits for: _uglobal and _ugroup for UAV
// memory, _g for group-shared memory, _t for the group's threads.
func syncFlags(f uint32) string {
	var s string
	for i, n := range []string{"_uglobal", "_ugroup", "_g", "_t"} {
		if f&(8>>i) != 0 {
			s += n
		}
	}
	return s
}

var conditional = map[Opcode]bool{
	OpIf: true, OpBreakc: true, OpContinuec: true, OpRetc: true, OpDiscard: true, OpCallc: true,
}

var mnemonicOverrides = map[Opcode]string{OpLdMs: "ldms", OpSampleInfo: "sampleinfo", OpSamplePos: "samplepos"}

// resourceDims names D3D10_SB_RESOURCE_DIMENSION as the disassembler does.
var resourceDims = [...]string{"unknown", "buffer", "texture1d", "texture2d", "texture2dms", "texture3d", "texturecube",
	"texture1darray", "texture2darray", "texture2dmsarray", "texturecubearray", "raw_buffer", "structured_buffer"}

var returnTypes = [...]string{"", "unorm", "snorm", "sint", "uint", "float", "mixed", "double", "continued"}

func resourceDim(d uint32) string {
	if int(d) < len(resourceDims) {
		return resourceDims[d]
	}
	return fmt.Sprintf("dim%d", d)
}

func returnTypeList(t uint32) string {
	parts := make([]string, 4)
	for i := range parts {
		r := t >> (4 * i) & 0xF
		if int(r) < len(returnTypes) && r != 0 {
			parts[i] = returnTypes[r]
		} else {
			parts[i] = fmt.Sprint(r)
		}
	}
	return "(" + strings.Join(parts, ",") + ")"
}

// extendedSuffix writes the extended opcode tokens: _aoffimmi(u,v,w) for
// immediate sample offsets, and _indexable(dimension)(return types) for the
// resource a sample, load or gather reads.
func extendedSuffix(in *Instruction) string {
	var offsets, dim, ret string
	for _, x := range in.Extended {
		switch x & 0x3F {
		case 1:
			u, v, w := signed4(x>>9), signed4(x>>13), signed4(x>>17)
			offsets = fmt.Sprintf("(%d,%d,%d)", u, v, w)
		case 2:
			dim = resourceDim(x >> 6 & 0x1F)
			if stride := x >> 11 & 0xFFF; stride != 0 {
				dim += fmt.Sprintf(", stride=%d", stride)
			}
		case 3:
			ret = returnTypeList(x >> 6)
		}
	}
	// ld_aoffimmi_indexable(1,0,0)(texture2d)(float,float,float,float)
	var s string
	if offsets != "" {
		s = "_aoffimmi"
	}
	if dim != "" || ret != "" {
		s += "_indexable"
	}
	s += offsets
	if dim != "" || ret != "" {
		s += "(" + dim + ")" + ret
	}
	return s
}
