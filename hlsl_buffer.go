package t7dxbc

import (
	"fmt"
	"math/bits"
	"strings"
)

// The buffers addressed by bytes: structured buffers and group memory by
// element and byte offset, raw buffers by byte offset.

// structuredDword is the expression of the dword c dwords past the byte
// offset offOp of element idx of a structured buffer or group memory, and
// its type. Group memory is dwords, so its offset may be computed; a
// buffer's must be a literal, which picks the member.
func (e *emitter) structuredDword(op *Operand, idx string, offOp *Operand, c uint32, store bool) (scalar, bool) {
	off, isImm := immediateScalar(offOp)
	if op.Type == OperandTGSM {
		g := uint32(op.Index[0].Imm)
		at := fmt.Sprint(off/4 + c)
		if !isImm {
			at = fmt.Sprintf("(%s >> 2) + %d", e.read(offOp, picks(offOp, 1), tUint), c)
		}
		expr := fmt.Sprintf("g%d[%s][%s]", g, idx, at)
		return scalar{expr: expr, base: expr, comp: -1, typ: TypeUint}, true
	}
	if !isImm {
		return e.dynamicDword(op, idx, offOp, c, store)
	}
	return e.structuredElement(op, idx, off+4*c)
}

// structuredElement is the expression of a structured buffer's dword at a
// byte offset of element idx, and its type.
func (e *emitter) structuredElement(op *Operand, idx string, off uint32) (scalar, bool) {
	res, b := e.resourceName(op)
	if b == nil {
		return scalar{}, false
	}
	t := e.structs[b.Name]
	if t == nil {
		return scalar{}, false
	}
	return scalarIn(fmt.Sprintf("%s[%s]", res, idx), t, off, false)
}

// ldStructuredStatement writes ld_structured: dest, index, byte offset,
// buffer.
func (e *emitter) ldStructuredStatement(in *Instruction, sat bool) {
	d, src := &in.Operands[0], &in.Operands[3]
	idx := e.read(&in.Operands[1], picks(&in.Operands[1], 1), tUint)
	cs := picks(src, destMask(d))
	ss := make([]scalar, len(cs))
	for i, c := range cs {
		s, ok := e.structuredDword(src, idx, &in.Operands[2], uint32(c), false)
		if !ok {
			e.unknown["ld_structured outside the element"] = true
			return
		}
		ss[i] = s
	}
	expr, t := joinScalars(ss, tUint)
	e.write(d, expr, t, sat)
}

// storeStructuredStatement writes store_structured: dest buffer, index, byte
// offset, value; one assignment per dword.
func (e *emitter) storeStructuredStatement(in *Instruction) {
	d := &in.Operands[0]
	idx := e.read(&in.Operands[1], picks(&in.Operands[1], 1), tUint)
	for c := range 4 {
		if destMask(d)&(1<<c) == 0 {
			continue
		}
		s, ok := e.structuredDword(d, idx, &in.Operands[2], uint32(c), true)
		if !ok {
			e.unknown["store_structured outside the element"] = true
			return
		}
		value := e.read(&in.Operands[3], picks(&in.Operands[3], 1<<c), varValType(s.typ))
		if s.store != "" {
			e.line(s.store+";", value)
			continue
		}
		e.line("%s = %s;", s.expr, value)
	}
}

// immediateScalar is an operand's value when it is a literal.
func immediateScalar(op *Operand) (uint32, bool) {
	if op.Type != OperandImmediate32 || len(op.Immediate) == 0 {
		return 0, false
	}
	c := 0
	if op.Selection == SelectOne || op.Selection == SelectSwizzle {
		c = int(op.Swizzle[0])
	}
	if c >= len(op.Immediate) {
		c = 0
	}
	return op.Immediate[c], true
}

// ldRawStatement writes ld_raw: dest, byte offset, raw buffer or group
// memory.
func (e *emitter) ldRawStatement(in *Instruction, sat bool) {
	d, src := &in.Operands[0], &in.Operands[2]
	addr := e.read(&in.Operands[1], picks(&in.Operands[1], 1), tUint)
	cs := picks(src, destMask(d))
	if src.Type == OperandTGSM {
		g := uint32(src.Index[0].Imm)
		parts := make([]string, len(cs))
		for i, c := range cs {
			parts[i] = fmt.Sprintf("g%d[(%s >> 2) + %d]", g, addr, c)
		}
		expr := parts[0]
		if len(parts) > 1 {
			expr = vecType(tUint, len(parts)) + "(" + strings.Join(parts, ", ") + ")"
		}
		e.write(d, expr, tUint, sat)
		return
	}
	res, b := e.resourceName(src)
	if b == nil {
		return
	}
	n := 0
	for _, c := range cs {
		n = max(n, int(c)+1)
	}
	method := "Load"
	if n > 1 {
		method += fmt.Sprint(n)
	}
	call := fmt.Sprintf("%s.%s(%s)", res, method, addr)
	if n > 1 {
		call += "." + letters(cs)
	}
	e.write(d, call, tUint, sat)
}

// storeRawStatement writes store_raw: dest buffer or group memory, byte
// offset, value.
func (e *emitter) storeRawStatement(in *Instruction) {
	d := &in.Operands[0]
	addr := e.read(&in.Operands[1], picks(&in.Operands[1], 1), tUint)
	mask := destMask(d)
	n := bits.OnesCount8(mask)
	if d.Type == OperandTGSM {
		g := uint32(d.Index[0].Imm)
		for i := range n {
			e.line("g%d[(%s >> 2) + %d] = %s;", g, addr, i, e.read(&in.Operands[2], picks(&in.Operands[2], 1<<i), tUint))
		}
		return
	}
	res, b := e.resourceName(d)
	if b == nil {
		return
	}
	method := "Store"
	if n > 1 {
		method += fmt.Sprint(n)
	}
	e.line("%s.%s(%s, %s);", res, method, addr, e.read(&in.Operands[2], picks(&in.Operands[2], mask), tUint))
}
