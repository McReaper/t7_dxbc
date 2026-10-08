package dxbc

import (
	"fmt"
	"math/bits"
	"strings"
)

// Registers hold bits, HLSL variables a type, and the compiler treats a
// float variable as a float: it flushes one holding a small integer's bits (a
// denormal) to zero while folding, and may lose a NaN's payload. So a
// temporary component that is ever written an integer lives in the integer
// bank, i# (uint4), and in the float bank r# otherwise; inputs and outputs
// are read and written as their parameters' own types. A float stored in i#
// is its exact bits.

// readTemp reads a temporary's components from their banks, as type t when
// they span both.
func (e *emitter) readTemp(op *Operand, cs []uint8, t valType) (string, valType) {
	reg := uint32(op.Index[0].Imm)
	ints := e.intRegs[reg]
	var nInt int
	for _, c := range cs {
		if ints&(1<<c) != 0 {
			nInt++
		}
	}
	switch nInt {
	case 0:
		return fmt.Sprintf("r%d.%s", reg, letters(cs)), tFloat
	case len(cs):
		return fmt.Sprintf("i%d.%s", reg, letters(cs)), tUint
	}
	parts := make([]string, len(cs))
	for i, c := range cs {
		if ints&(1<<c) != 0 {
			parts[i] = convert(fmt.Sprintf("i%d.%c", reg, components[c]), tUint, t)
		} else {
			parts[i] = convert(fmt.Sprintf("r%d.%c", reg, components[c]), tFloat, t)
		}
	}
	return vecType(t, len(cs)) + "(" + strings.Join(parts, ", ") + ")", t
}

// writeTemp assigns an expression of type t to a temporary's components, in
// their banks; while the banks are being found, it records which components
// receive integers.
func (e *emitter) writeTemp(dst *Operand, expr string, t valType) {
	reg := uint32(dst.Index[0].Imm)
	mask := destMask(dst)
	if e.dry {
		if t != tFloat {
			e.intRegs[reg] |= mask
		}
		return
	}
	ints := e.intRegs[reg] & mask
	floats := mask &^ ints
	if ints == 0 || floats == 0 {
		bank, bt := "r", tFloat
		if floats == 0 {
			bank, bt = "i", tUint
		}
		e.line("%s%d.%s = %s;", bank, reg, maskLetters(mask), convert(expr, t, bt))
		return
	}
	e.split(expr, t, mask, []split{
		{fmt.Sprintf("r%d.%s", reg, maskLetters(floats)), floats, tFloat},
		{fmt.Sprintf("i%d.%s", reg, maskLetters(ints)), ints, tUint},
	})
}

// split is a part of a value written to several places: where it goes, the
// components of the written mask it takes, and the type there.
type split struct {
	target string
	part   uint8
	t      valType
}

// split evaluates an expression once and assigns its parts: the components
// of mask that part holds, as the vector's components in mask order.
func (e *emitter) split(expr string, t valType, mask uint8, parts []split) {
	e.line("{")
	e.depth++
	e.line("%s dxbc_v = %s;", vecType(t, bits.OnesCount8(mask)), expr)
	for _, p := range parts {
		e.line("%s = %s;", p.target, convert("dxbc_v."+within(mask, p.part), t, p.t))
	}
	e.depth--
	e.line("}")
}

// within lists, as letters, where the components of part sit in a vector of
// mask's components: xz within xyz is xz, z within yz is y.
func within(mask, part uint8) string {
	var sb strings.Builder
	for c := range 4 {
		if part&(1<<c) != 0 {
			sb.WriteByte(components[bits.OnesCount8(mask&(1<<c-1))])
		}
	}
	return sb.String()
}

// elementAt finds the signature element holding component c of a register,
// and the letter of that component in the element.
func elementAt(els []Element, reg uint32, c uint8) (Element, byte, bool) {
	for _, el := range els {
		if el.Register == reg && el.Mask&(1<<c) != 0 {
			return el, components[bits.OnesCount8(el.Mask&(1<<c-1))], true
		}
	}
	return Element{}, 0, false
}

func componentValType(t ComponentType) valType {
	switch t {
	case ComponentUint:
		return tUint
	case ComponentSint:
		return tInt
	}
	return tFloat
}

// readInput reads an input register's components from the parameters they
// arrived in.
func (e *emitter) readInput(op *Operand, cs []uint8, t valType) (string, valType) {
	if len(op.Index) != 1 || op.Index[0].Relative != nil {
		e.unknown["indexed input"] = true
		return "0", t
	}
	reg := uint32(op.Index[0].Imm)
	type part struct {
		name   string
		letter byte
		t      valType
	}
	parts := make([]part, len(cs))
	for i, c := range cs {
		el, letter, ok := elementAt(e.in, reg, c)
		if !ok {
			e.unknown[fmt.Sprintf("v%d.%c outside the input signature", reg, components[c])] = true
			return "0", t
		}
		name, pt := paramName("in", el), componentValType(el.Type)
		if strings.EqualFold(el.Name, "SV_IsFrontFace") {
			name, pt = "("+name+" ? 0xffffffffu : 0u)", tUint
		}
		parts[i] = part{name, letter, pt}
	}
	same := true
	for _, p := range parts[1:] {
		if p.name != parts[0].name {
			same = false
		}
	}
	if same {
		ls := make([]byte, len(parts))
		for i, p := range parts {
			ls[i] = p.letter
		}
		return parts[0].name + "." + string(ls), parts[0].t
	}
	nt := parts[0].t // one type for all: no conversion needed
	for _, p := range parts[1:] {
		if p.t != nt {
			nt = t
		}
	}
	exprs := make([]string, len(parts))
	for i, p := range parts {
		exprs[i] = convert(fmt.Sprintf("%s.%c", p.name, p.letter), p.t, nt)
	}
	return vecType(nt, len(parts)) + "(" + strings.Join(exprs, ", ") + ")", nt
}

// writeOutput assigns an expression to an output register's components, in
// the parameters they leave through.
func (e *emitter) writeOutput(dst *Operand, expr string, t valType) {
	if len(dst.Index) != 1 || dst.Index[0].Relative != nil {
		e.unknown["indexed output"] = true
		return
	}
	if e.dry {
		return
	}
	reg := uint32(dst.Index[0].Imm)
	mask := destMask(dst)
	var parts []split
	for c := range uint8(4) {
		if mask&(1<<c) == 0 {
			continue
		}
		el, letter, ok := elementAt(e.out, reg, c)
		if !ok {
			e.unknown[fmt.Sprintf("o%d.%c outside the output signature", reg, components[c])] = true
			return
		}
		name := paramName("out", el)
		if n := len(parts); n > 0 && strings.HasPrefix(parts[n-1].target, name+".") {
			parts[n-1].target += string(letter)
			parts[n-1].part |= 1 << c
			continue
		}
		parts = append(parts, split{name + "." + string(letter), 1 << c, componentValType(el.Type)})
	}
	if len(parts) == 1 {
		e.line("%s = %s;", parts[0].target, convert(expr, t, parts[0].t))
		return
	}
	e.split(expr, t, mask, parts)
}

// readAny reads an operand in the type it has, for the instructions that move
// bits whatever they are: mov, movc, swapc. A modifier makes them float
// operations.
func (e *emitter) readAny(op *Operand, cs []uint8) (string, valType) {
	if op.Modifier != ModNone {
		return e.read(op, cs, tFloat), tFloat
	}
	if op.Type == OperandImmediate32 {
		t := tFloat
		for _, c := range cs {
			if c < uint8(len(op.Immediate)) && looksInteger(op.Immediate[c]) {
				t = tUint
			}
		}
		return immediate(op, cs, t), t
	}
	return e.readRaw(op, cs, tUint)
}

// looksInteger tells an integer's bits from a float's as the disassembler
// does: a float with a zero or all-ones exponent is a denormal, an infinity
// or a NaN, which shaders rarely hold, and small integers have one.
func looksInteger(v uint32) bool {
	exp := v >> 23 & 0xFF
	return v != 0 && v != 0x80000000 && (exp == 0 || exp == 0xFF)
}

// readIndexable reads an indexable temporary, an array of float4 or of uint4.
func (e *emitter) readIndexable(op *Operand, cs []uint8) (string, valType) {
	x := uint32(op.Index[0].Imm)
	t := tFloat
	if e.intX[x] {
		t = tUint
	}
	return fmt.Sprintf("x%d[%s].%s", x, e.indexExpr(op.Index[1]), letters(cs)), t
}

// writeIndexable assigns to an indexable temporary.
func (e *emitter) writeIndexable(dst *Operand, expr string, t valType) {
	x := uint32(dst.Index[0].Imm)
	if e.dry {
		if t != tFloat {
			e.intX[x] = true
		}
		return
	}
	bt := tFloat
	if e.intX[x] {
		bt = tUint
	}
	e.line("x%d[%s].%s = %s;", x, e.indexExpr(dst.Index[1]), maskLetters(destMask(dst)), convert(expr, t, bt))
}
