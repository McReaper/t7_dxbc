package dxbc

import (
	"fmt"
	"math"
	"strings"
)

// operandPrefixes name the registers of each operand type the way the
// disassembler does; a type with a fixed name has no number.
var operandPrefixes = map[OperandType]string{
	OperandTemp: "r", OperandInput: "v", OperandOutput: "o", OperandIndexableTemp: "x",
	OperandSampler: "s", OperandResource: "t", OperandCBuffer: "cb", OperandICB: "icb", OperandLabel: "l", 14: "rasterizer",
	OperandNull: "null", OperandUAV: "u", OperandTGSM: "g",
	11: "vPrim", 12: "oDepth", 15: "oMask", 16: "m", 17: "fb", 18: "ft", 19: "fp", 22: "vOutputControlPointID",
	23: "vForkInstanceID", 24: "vJoinInstanceID", 25: "vicp", 26: "vocp", 27: "vpc", 28: "vDomain", 29: "this",
	32: "vThreadID", 33: "vThreadGroupID", 34: "vThreadIDInGroup", 35: "vCoverage", 36: "vThreadIDInGroupFlattened",
	37: "vGSInstanceID", 38: "oDepthGE", 39: "oDepthLE", 40: "vCycleCounter", 41: "oStencilRef", 42: "vInnerCoverage",
}

// numberless are the operand types whose index, if any, isn't a register
// number written after the name.
var numberless = map[OperandType]bool{
	OperandNull: true, OperandICB: true, 11: true, 14: true, 12: true, 15: true, 22: true, 23: true, 24: true, 28: true,
	32: true, 33: true, 34: true, 35: true, 36: true, 37: true, 38: true, 39: true, 40: true, 41: true, 42: true,
}

// numberStyle is how an instruction's immediates are written: as floats, as
// integers, or by what the bits look like.
type numberStyle uint8

const (
	numTypeless numberStyle = iota
	numFloat
	numInt
	numUint
)

const components = "xyzw"

// Prefix names an operand type's register file as the disassembler writes
// it: r, v, o, cb, vThreadID…
func (t OperandType) Prefix() string { return operandPrefixes[t] }

// Disassemble writes an operand as the disassembler does, its immediates
// without a type.
func (op *Operand) Disassemble() string { return formatOperand(op, numTypeless) }

// formatOperand writes an operand as the disassembler does.
func formatOperand(op *Operand, style numberStyle) string {
	var sb strings.Builder
	switch op.Modifier {
	case ModNeg:
		sb.WriteByte('-')
	case ModAbs, ModAbsNeg:
		if op.Modifier == ModAbsNeg {
			sb.WriteByte('-')
		}
		sb.WriteByte('|')
	}
	sb.WriteString(operandBody(op, style))
	if op.Modifier == ModAbs || op.Modifier == ModAbsNeg {
		sb.WriteByte('|')
	}
	return sb.String()
}

func operandBody(op *Operand, style numberStyle) string {
	if op.Type == OperandImmediate32 || op.Type == OperandImmediate64 {
		return "l(" + formatImmediates(op, style) + ")"
	}
	name := operandPrefixes[op.Type]
	if name == "" {
		name = fmt.Sprintf("operand%d_", op.Type)
	}
	idx := op.Index
	switch {
	case numberless[op.Type]:
	case op.Type == OperandInput && len(idx) == 2: // a geometry shader's vertex and register: v[0][3]
	case len(idx) > 0 && idx[0].Relative == nil:
		name += fmt.Sprint(idx[0].Imm)
		idx = idx[1:]
	}
	for _, i := range idx {
		name += "[" + formatIndex(i) + "]"
	}
	return name + selection(op)
}

func formatIndex(i Index) string {
	if i.Relative == nil {
		return fmt.Sprint(i.Imm)
	}
	return fmt.Sprintf("%s + %d", formatOperand(i.Relative, numInt), i.Imm)
}

// selection writes the components an operand picks: a mask names those it
// writes, a swizzle all four, a single selection one.
func selection(op *Operand) string {
	if op.Components != 4 {
		return ""
	}
	var sb strings.Builder
	sb.WriteByte('.')
	switch op.Selection {
	case SelectMask:
		if op.Mask == 0 {
			return ""
		}
		for i := range 4 {
			if op.Mask&(1<<i) != 0 {
				sb.WriteByte(components[i])
			}
		}
	case SelectSwizzle:
		for _, c := range op.Swizzle {
			sb.WriteByte(components[c])
		}
	case SelectOne:
		sb.WriteByte(components[op.Swizzle[0]])
	}
	return sb.String()
}

// formatImmediates writes an immediate's values: a typed instruction's apart
// with ", ", a typeless one's (mov) with "," alone.
func formatImmediates(op *Operand, style numberStyle) string {
	parts := make([]string, len(op.Immediate))
	for i, v := range op.Immediate {
		parts[i] = formatNumber(v, style)
	}
	if style == numTypeless {
		return strings.Join(parts, ",")
	}
	return strings.Join(parts, ", ")
}

// formatNumber writes one 32-bit immediate in the instruction's style; a
// typeless one is a float unless its bits are a zero, a denormal, an infinity
// or a NaN, which a float rarely is and an integer often.
func formatNumber(v uint32, style numberStyle) string {
	switch style {
	case numFloat:
		return formatFloat(v)
	case numInt, numUint:
		return formatInt(v)
	}
	if exp := v >> 23 & 0xFF; (exp == 0 || exp == 0xFF) && v != 0x80000000 { // -0.0 reads as a float
		return formatInt(v)
	}
	return formatFloat(v)
}

// formatInt writes an integer in decimal from -10000 to 10000, and beyond as
// its 32 bits in hex.
func formatInt(v uint32) string {
	if s := int32(v); s >= -10000 && s <= 10000 {
		return fmt.Sprint(s)
	}
	return fmt.Sprintf("0x%08x", v)
}

// formatFloat writes a float with %f; infinities and NaNs as the MSVC
// runtime's %f writes them (the corpus has infinities, no NaN).
func formatFloat(v uint32) string {
	f := math.Float32frombits(v)
	switch {
	case math.IsInf(float64(f), 1):
		return "1.#INF00"
	case math.IsInf(float64(f), -1):
		return "-1.#INF00"
	case v == 0xFFC00000:
		return "-1.#IND00"
	case f != f && v>>31 != 0:
		return "-1.#QNAN0"
	case f != f:
		return "1.#QNAN0"
	}
	return fmt.Sprintf("%f", float64(f))
}
