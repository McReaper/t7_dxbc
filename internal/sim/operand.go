package sim

import (
	"fmt"
	"math"

	"github.com/McReaper/dxbc"
)

// src reads a source operand's four components through its swizzle, without
// its modifier, which depends on the instruction's type.
func (t *thread) src(op *dxbc.Operand) Value {
	v := t.load(op)
	switch {
	case op.Components == 1:
		return Value{v[0], v[0], v[0], v[0]}
	case op.Selection == dxbc.SelectSwizzle:
		return Value{v[op.Swizzle[0]&3], v[op.Swizzle[1]&3], v[op.Swizzle[2]&3], v[op.Swizzle[3]&3]}
	case op.Selection == dxbc.SelectOne:
		c := v[op.Swizzle[0]&3]
		return Value{c, c, c, c}
	}
	return v
}

// index is one of an operand's indices: its immediate plus its relative
// register's value.
func (t *thread) index(op *dxbc.Operand, i int) int64 {
	if i >= len(op.Index) {
		return 0
	}
	n := int64(op.Index[i].Imm)
	if r := op.Index[i].Relative; r != nil {
		n += int64(int32(t.src(r)[0]))
	}
	return n
}

// load reads a register's four components, before any swizzle. What lies out
// of range reads 0, as on the hardware.
func (t *thread) load(op *dxbc.Operand) Value {
	switch op.Type {
	case dxbc.OperandTemp:
		if i := t.index(op, 0); i >= 0 && i < int64(len(t.r)) {
			return t.r[i]
		}
	case dxbc.OperandInput:
		if len(op.Index) == 2 { // a geometry shader's v[vertex][register]
			return t.in[VertexInput(uint32(t.index(op, 0)), uint32(t.index(op, 1)))]
		}
		return t.in[uint32(t.index(op, 0))]
	case dxbc.OperandIndexableTemp:
		arr := t.x[uint32(t.index(op, 0))]
		if i := t.index(op, 1); i >= 0 && i < int64(len(arr)) {
			return arr[i]
		}
	case dxbc.OperandImmediate32:
		var v Value
		copy(v[:], op.Immediate)
		return v
	case dxbc.OperandCBuffer:
		return dwords(t.w.CB[uint32(t.index(op, 0))], t.index(op, 1)*4)
	case dxbc.OperandICB:
		return dwords(t.m.icb, t.index(op, 0)*4)
	default:
		return t.sys[op.Type]
	}
	return Value{}
}

// dwords reads four dwords from at, or 0 past the end.
func dwords(d []uint32, at int64) Value {
	var v Value
	for i := range v {
		if j := at + int64(i); j >= 0 && j < int64(len(d)) {
			v[i] = d[j]
		}
	}
	return v
}

// mask is the components a destination writes.
func mask(op *dxbc.Operand) uint8 {
	if op.Components == 4 && op.Selection == dxbc.SelectMask {
		return op.Mask
	}
	return 1
}

// store writes the destination's components of v, saturated if asked.
func (t *thread) store(op *dxbc.Operand, v Value, sat bool) error {
	if sat {
		for i := range v {
			v[i] = saturate(v[i])
		}
	}
	m := mask(op)
	put := func(dst *Value) {
		for i := range dst {
			if m&(1<<i) != 0 {
				dst[i] = v[i]
			}
		}
	}
	switch op.Type {
	case dxbc.OperandNull:
	case dxbc.OperandTemp:
		i := t.index(op, 0)
		if i < 0 || i >= int64(len(t.r)) {
			return fmt.Errorf("r%d out of range", i)
		}
		put(&t.r[i])
	case dxbc.OperandOutput:
		o := t.res.Outputs[uint32(t.index(op, 0))]
		put(&o)
		t.res.Outputs[uint32(t.index(op, 0))] = o
	case dxbc.OperandIndexableTemp:
		arr := t.x[uint32(t.index(op, 0))]
		if i := t.index(op, 1); i >= 0 && i < int64(len(arr)) {
			put(&arr[i])
		}
	default:
		if op.Components > 1 {
			return fmt.Errorf("destination of type %d", op.Type)
		}
		t.res.System[op.Type] = v[0]
	}
	return nil
}

// saturate clamps a float to [0, 1]; NaN becomes 0.
func saturate(b uint32) uint32 {
	f := math.Float32frombits(b)
	switch {
	case f != f || f <= 0:
		return 0
	case f >= 1:
		return math.Float32bits(1)
	}
	return b
}

// fsrc reads a float operand with its modifier, denormals flushed to zero.
func (t *thread) fsrc(op *dxbc.Operand) [4]float32 {
	v := t.src(op)
	var f [4]float32
	for i, b := range v {
		b = flush(b)
		switch op.Modifier {
		case dxbc.ModAbs:
			b &^= 1 << 31
		case dxbc.ModNeg:
			b ^= 1 << 31
		case dxbc.ModAbsNeg:
			b |= 1 << 31
		}
		f[i] = math.Float32frombits(b)
	}
	return f
}

// isrc reads an integer operand with its modifier: neg is the two's
// complement.
func (t *thread) isrc(op *dxbc.Operand) Value {
	v := t.src(op)
	if op.Modifier == dxbc.ModNeg {
		for i := range v {
			v[i] = -v[i]
		}
	}
	return v
}

// flush makes a denormal zero, keeping its sign, as float arithmetic does.
func flush(b uint32) uint32 {
	if b>>23&0xFF == 0 {
		return b & (1 << 31)
	}
	return b
}

// fbits is a float result's bits, a denormal flushed.
func fbits(f float32) uint32 { return flush(math.Float32bits(f)) }
