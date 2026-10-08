package dxbc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// valType is the type an expression has, or an operation reads its operands
// as.
type valType uint8

const (
	tFloat valType = iota
	tInt
	tUint
)

func (t valType) String() string { return [...]string{"float", "int", "uint"}[t] }

// vecType spells an n-component vector of the type: float, int3.
func vecType(t valType, n int) string {
	if n == 1 {
		return t.String()
	}
	return fmt.Sprintf("%s%d", t, n)
}

// convert reinterprets an expression's bits as another type: asfloat, asint,
// asuint. Never a cast: (int) of a uint2 would keep only its x.
func convert(expr string, from, to valType) string {
	switch {
	case from == to:
		return expr
	case to == tFloat:
		return "asfloat(" + expr + ")"
	case to == tInt:
		return "asint(" + expr + ")"
	}
	return "asuint(" + expr + ")"
}

// picks lists the source components an operand reads for each written
// component of mask.
func picks(op *Operand, mask uint8) []uint8 {
	var out []uint8
	for i := range 4 {
		if mask&(1<<i) == 0 {
			continue
		}
		switch op.Selection {
		case SelectOne:
			out = append(out, op.Swizzle[0])
		case SelectSwizzle:
			out = append(out, op.Swizzle[i])
		default:
			out = append(out, uint8(i))
		}
	}
	if op.Components == 1 || op.Components == 0 {
		out = make([]uint8, len(out)) // a scalar operand reads its one value
	}
	return out
}

// first lists the first n components an operand's swizzle reads, as a dot
// product or an address does whatever the destination.
func first(op *Operand, n int) []uint8 {
	var mask uint8 = 1<<n - 1
	if op.Selection == SelectOne {
		return picks(op, mask)
	}
	out := make([]uint8, n)
	for i := range out {
		if op.Selection == SelectSwizzle {
			out[i] = op.Swizzle[i]
		} else {
			out[i] = uint8(i)
		}
	}
	return out
}

func letters(cs []uint8) string {
	b := make([]byte, len(cs))
	for i, c := range cs {
		b[i] = components[c]
	}
	return string(b)
}

// read is an operand's value at the components cs, as type t, with its
// modifiers applied.
func (e *emitter) read(op *Operand, cs []uint8, t valType) string {
	expr, natural := e.readRaw(op, cs, t)
	expr = convert(expr, natural, t)
	switch op.Modifier {
	case ModNeg:
		expr = "-" + paren(expr)
	case ModAbs:
		expr = "abs(" + expr + ")"
	case ModAbsNeg:
		expr = "-abs(" + expr + ")"
	}
	return expr
}

// paren wraps an expression that isn't a plain name, call or literal.
func paren(expr string) string {
	if strings.HasPrefix(expr, "-") {
		return "(" + expr + ")"
	}
	for _, r := range expr {
		if r == ' ' || r == '?' {
			return "(" + expr + ")"
		}
	}
	return expr
}

// readRaw is an operand's value at the components cs, in the type it
// naturally has.
func (e *emitter) readRaw(op *Operand, cs []uint8, t valType) (string, valType) {
	sw := "." + letters(cs)
	switch op.Type {
	case OperandTemp:
		if len(op.Index) == 1 && op.Index[0].Relative == nil {
			return e.readTemp(op, cs, t)
		}
	case OperandInput:
		return e.readInput(op, cs, t)
	case OperandIndexableTemp:
		if len(op.Index) == 2 {
			return e.readIndexable(op, cs)
		}
	case OperandImmediate32:
		return immediate(op, cs, t), t
	case OperandCBuffer:
		return e.readCBuffer(op, cs, t)
	case OperandICB:
		if len(op.Index) == 0 {
			break
		}
		return fmt.Sprintf("icb[%s]%s", e.indexExpr(op.Index[0]), sw), tUint
	}
	if st, ok := systemInputs[op.Type]; ok {
		if st[0] == "uint" {
			return operandPrefixes[op.Type], tUint
		}
		return operandPrefixes[op.Type] + sw, tUint
	}
	if _, ok := systemOutputs[op.Type]; ok {
		return operandPrefixes[op.Type], tFloat
	}
	e.unknown[fmt.Sprintf("operand %s", operandPrefixes[op.Type])] = true
	return "0", t
}

// indexExpr is an index as an int expression: 3, asint(r1.y) + 2.
func (e *emitter) indexExpr(i Index) string {
	if i.Relative == nil {
		return fmt.Sprint(i.Imm)
	}
	rel := e.read(i.Relative, picks(i.Relative, 1), tInt)
	if i.Imm == 0 {
		return rel
	}
	return fmt.Sprintf("%s + %d", rel, i.Imm)
}

// immediate writes the literal values at cs in type t.
func immediate(op *Operand, cs []uint8, t valType) string {
	vals := make([]string, len(cs))
	for i, c := range cs {
		v := op.Immediate[0]
		if int(c) < len(op.Immediate) {
			v = op.Immediate[c]
		}
		vals[i] = literal(v, t)
	}
	if len(vals) == 1 {
		return vals[0]
	}
	return vecType(t, len(vals)) + "(" + strings.Join(vals, ", ") + ")"
}

// literal writes a 32-bit value as an exact literal of type t; a float that
// isn't a normal number is its bits.
func literal(v uint32, t valType) string {
	switch t {
	case tInt:
		return fmt.Sprint(int32(v))
	case tUint:
		if v > 0xFFFF {
			return fmt.Sprintf("0x%08xu", v)
		}
		return fmt.Sprintf("%du", v)
	}
	if v == 0 {
		return "0"
	}
	exp := v >> 23 & 0xFF
	if exp == 0 || exp == 0xFF {
		return fmt.Sprintf("asfloat(0x%08xu)", v)
	}
	s := strconv.FormatFloat(float64(math.Float32frombits(v)), 'g', -1, 32)
	if !strings.ContainsAny(s, ".e") {
		s += ".0"
	}
	return s
}

// readCBuffer reads constant buffer components by the variables that hold
// them.
func (e *emitter) readCBuffer(op *Operand, cs []uint8, t valType) (string, valType) {
	if len(op.Index) < 2 {
		e.unknown["cb operand"] = true
		return "0", t
	}
	l := e.cbuffers[uint32(op.Index[0].Imm)]
	if l == nil {
		e.unknown[fmt.Sprintf("cb%d without reflection", op.Index[0].Imm)] = true
		return "0", t
	}
	if op.Index[1].Relative != nil {
		return e.readCBufferDynamic(l, op, cs, t)
	}
	reg := uint32(op.Index[1].Imm)
	ss := make([]scalar, len(cs))
	for i, c := range cs {
		s, ok := l.at(reg*16 + uint32(c)*4)
		if !ok {
			e.unknown[fmt.Sprintf("%s offset %d", l.buf.Name, reg*16+uint32(c)*4)] = true
			return "0", t
		}
		ss[i] = s
	}
	return joinScalars(ss, t)
}

// joinScalars reads several scalars as one vector: a swizzle of their vector
// when they share one, else a constructor of each converted to t.
func joinScalars(ss []scalar, t valType) (string, valType) {
	same := ss[0].comp >= 0
	for _, s := range ss[1:] {
		if s.base != ss[0].base || s.comp < 0 || s.typ != ss[0].typ {
			same = false
		}
	}
	nat := varValType(ss[0].typ)
	if same {
		cs := make([]uint8, len(ss))
		for i, s := range ss {
			cs[i] = uint8(s.comp)
		}
		return ss[0].base + "." + letters(cs), nat
	}
	if len(ss) == 1 {
		return ss[0].expr, nat
	}
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = convert(s.expr, varValType(s.typ), t)
	}
	return vecType(t, len(ss)) + "(" + strings.Join(parts, ", ") + ")", t
}

func varValType(v VarType) valType {
	switch v {
	case TypeInt, TypeMin12Int, TypeMin16Int:
		return tInt
	case TypeUint, TypeUint8, TypeBool, TypeMin16Uint:
		return tUint
	}
	return tFloat
}

// readCBufferDynamic reads an array element at a register index computed at
// run time: cb0[r1.y + 20] in an array of structs three registers long, at
// register 18, is element r1.y / 3, and its member at register 2. The array
// may be a struct's member: the first on the way in takes the index.
func (e *emitter) readCBufferDynamic(l *cbufferLayout, op *Operand, cs []uint8, t valType) (string, valType) {
	reg := uint32(op.Index[1].Imm)
	fail := func() (string, valType) {
		e.unknown[fmt.Sprintf("dynamic index into %s at register %d", l.buf.Name, reg)] = true
		return "0", t
	}
	for _, v := range l.vars {
		if reg*16 < v.Offset || reg*16 >= v.Offset+v.Size {
			continue
		}
		name, elem, inner, ok := e.dynamicElement(identifier(v.Name), v.Type, reg*16-v.Offset, op.Index[1].Relative)
		if !ok {
			break
		}
		ss := make([]scalar, len(cs))
		for i, c := range cs {
			if ss[i], ok = scalarIn(name, elem, inner+uint32(c)*4, true); !ok {
				return fail()
			}
		}
		return joinScalars(ss, t)
	}
	return fail()
}

// dynamicElement finds the array a dynamic register index at off bytes into
// a value of type t selects an element of: that element's expression and
// type, and the offset in it.
func (e *emitter) dynamicElement(name string, t *Type, off uint32, rel *Operand) (string, *Type, uint32, bool) {
	if t.Elements > 0 {
		elem := *t
		elem.Elements = 0
		stride := (typeSize(&elem, true) + 15) &^ 15
		var idx string
		if regs := stride / 16; regs == 1 {
			idx = e.read(rel, picks(rel, 1), tInt)
		} else {
			idx = fmt.Sprintf("%s / %du", paren(e.read(rel, picks(rel, 1), tUint)), regs)
		}
		if k := off / stride; k != 0 {
			idx = fmt.Sprintf("%s + %d", idx, k)
		}
		return fmt.Sprintf("%s[%s]", name, idx), &elem, off % stride, true
	}
	if t.Class == ClassStruct {
		for i := len(t.Members) - 1; i >= 0; i-- {
			if m := t.Members[i]; off >= m.Offset {
				return e.dynamicElement(name+"."+identifier(m.Name), m.Type, off-m.Offset, rel)
			}
		}
	}
	return "", nil, 0, false
}

// write assigns an expression of type t to a destination, or does nothing
// for a null one.
func (e *emitter) write(dst *Operand, expr string, t valType, sat bool) {
	if dst.Type == OperandNull {
		return
	}
	if sat {
		expr = "saturate(" + convert(expr, t, tFloat) + ")"
		t = tFloat
	}
	switch {
	case dst.Type == OperandTemp && len(dst.Index) == 1 && dst.Index[0].Relative == nil:
		e.writeTemp(dst, expr, t)
	case dst.Type == OperandOutput:
		e.writeOutput(dst, expr, t)
	case dst.Type == OperandIndexableTemp && len(dst.Index) == 2:
		e.writeIndexable(dst, expr, t)
	default:
		if out, ok := systemOutputs[dst.Type]; ok {
			nat := tFloat
			if out[0] == "uint" {
				nat = tUint
			}
			if !e.dry {
				e.line("%s = %s;", operandPrefixes[dst.Type], convert(expr, t, nat))
			}
			return
		}
		e.unknown[fmt.Sprintf("destination %s", operandPrefixes[dst.Type])] = true
	}
}

// destMask is the components a destination writes; a scalar destination
// writes x.
func destMask(dst *Operand) uint8 {
	if dst.Components == 4 && dst.Selection == SelectMask && dst.Mask != 0 {
		return dst.Mask
	}
	return 1
}
