package sim

import (
	"fmt"

	"github.com/McReaper/t7dxbc/dxbc"
)

// word is one dword of a buffer or of group-shared memory, read and written.
type word struct {
	get func() uint32
	set func(uint32)
}

// word finds the dword a structured or raw access reaches: element index
// and byte offset for a structured view or group memory, a byte address for a
// raw one. Typed UAVs are reached by coordinates instead.
func (t *thread) word(op *dxbc.Operand, index, byteOff uint32) (word, error) {
	reg := uint32(t.index(op, 0))
	if op.Type == dxbc.OperandTGSM {
		d := t.m.tgsm[reg]
		mem := t.g[reg]
		at := byteOff / 4
		if d[0] != 0 {
			at = (index*d[0] + byteOff) / 4
		}
		if at >= uint32(len(mem)) {
			return word{func() uint32 { return 0 }, func(uint32) {}}, nil
		}
		return word{func() uint32 { return mem[at] }, func(v uint32) { mem[at] = v }}, nil
	}
	b, err := t.binding(op)
	if err != nil {
		return word{}, err
	}
	v := view{op.Type, reg}
	stride, structured := t.m.strides[v]
	inRange := index < bufElements && (!structured || byteOff < stride) && (structured || byteOff < bufElements*64)
	isFloat := false
	if structured {
		isFloat = memberIsFloat(t.m.elems[v], byteOff)
	}
	mock := func() uint32 {
		if !inRange {
			return 0
		}
		return t.w.hashed(isFloat, uint32(op.Type), reg, index, byteOff)
	}
	if op.Type != dxbc.OperandUAV {
		return word{mock, func(uint32) {}}, nil
	}
	if b.Input == dxbc.InputUAVTyped {
		return word{}, fmt.Errorf("u%d is typed", reg)
	}
	c := Cell{Reg: reg, A: index, B: byteOff / 4}
	return word{
		get: func() uint32 {
			if x, ok := t.w.UAV[c]; ok {
				return x
			}
			return mock()
		},
		set: func(x uint32) {
			if inRange {
				t.w.UAV[c] = x
			}
		},
	}, nil
}

// memberIsFloat reports whether the dword at off in a structured element is
// a float, the element laid out tightly.
func memberIsFloat(t *dxbc.Type, off uint32) bool {
	if t == nil {
		return false
	}
	if t.Elements > 0 {
		e := *t
		e.Elements = 0
		if s := tightSize(&e); s > 0 {
			return memberIsFloat(&e, off%s)
		}
		return false
	}
	if t.Class == dxbc.ClassStruct {
		for i := len(t.Members) - 1; i >= 0; i-- {
			if off >= t.Members[i].Offset {
				return memberIsFloat(t.Members[i].Type, off-t.Members[i].Offset)
			}
		}
		return false
	}
	return t.Base == dxbc.TypeFloat
}

// tightSize is a type's size in a structured buffer.
func tightSize(t *dxbc.Type) uint32 {
	var one uint32
	switch t.Class {
	case dxbc.ClassScalar:
		one = 4
	case dxbc.ClassVector:
		one = 4 * uint32(t.Cols)
	case dxbc.ClassMatrixRows, dxbc.ClassMatrixColumns:
		one = 4 * uint32(t.Rows) * uint32(t.Cols)
	case dxbc.ClassStruct:
		for _, m := range t.Members {
			one = max(one, m.Offset+tightSize(m.Type))
		}
	}
	return one * uint32(max(t.Elements, 1))
}

func ldStructured(t *thread, in *dxbc.Instruction) error {
	idx, off := t.src(&in.Operands[1])[0], t.src(&in.Operands[2])[0]
	var v Value
	for i := range v {
		w, err := t.word(&in.Operands[3], idx, off+4*uint32(i))
		if err != nil {
			return err
		}
		v[i] = w.get()
	}
	return t.storeFetched(in, &in.Operands[3], v, false)
}

func ldRaw(t *thread, in *dxbc.Instruction) error {
	addr := t.src(&in.Operands[1])[0]
	var v Value
	for i := range v {
		w, err := t.word(&in.Operands[2], 0, addr+4*uint32(i))
		if err != nil {
			return err
		}
		v[i] = w.get()
	}
	return t.storeFetched(in, &in.Operands[2], v, false)
}

func storeStructured(t *thread, in *dxbc.Instruction) error {
	dst := &in.Operands[0]
	idx, off, v := t.src(&in.Operands[1])[0], t.src(&in.Operands[2])[0], t.src(&in.Operands[3])
	return t.storeWords(dst, v, func(c uint32) (word, error) { return t.word(dst, idx, off+4*c) })
}

func storeRaw(t *thread, in *dxbc.Instruction) error {
	dst := &in.Operands[0]
	addr, v := t.src(&in.Operands[1])[0], t.src(&in.Operands[2])
	return t.storeWords(dst, v, func(c uint32) (word, error) { return t.word(dst, 0, addr+4*c) })
}

// storeWords writes the destination mask's components of v, each to its
// dword.
func (t *thread) storeWords(dst *dxbc.Operand, v Value, at func(c uint32) (word, error)) error {
	m := mask(dst)
	for c := range uint32(4) {
		if m&(1<<c) == 0 {
			continue
		}
		w, err := at(c)
		if err != nil {
			return err
		}
		w.set(v[c])
	}
	return nil
}

// atomicWord finds the dword an atomic operation changes: a typed UAV by
// coordinates, a raw view by byte address, a structured one by index and
// offset.
func (t *thread) atomicWord(op *dxbc.Operand, addr Value) (word, error) {
	if op.Type == dxbc.OperandUAV {
		b, err := t.binding(op)
		if err != nil {
			return word{}, err
		}
		if b.Input == dxbc.InputUAVTyped {
			var key [4]uint32
			copy(key[:], addr[:uavCoordCount(b.Dim)])
			c := Cell{uint32(t.index(op, 0)), key[0], key[1], key[2], 0}
			return word{
				get: func() uint32 {
					if x, ok := t.w.UAV[c]; ok {
						return x
					}
					return t.fetchTyped(op, b, key)[0]
				},
				set: func(x uint32) { t.w.UAV[c] = x },
			}, nil
		}
	}
	if t.isStructured(op) {
		return t.word(op, addr[0], addr[1])
	}
	return t.word(op, 0, addr[0])
}

func (t *thread) isStructured(op *dxbc.Operand) bool {
	if op.Type == dxbc.OperandTGSM {
		return t.m.tgsm[uint32(t.index(op, 0))][0] != 0
	}
	_, ok := t.m.strides[view{op.Type, uint32(t.index(op, 0))}]
	return ok
}

// atomicFns are what each atomic operation makes of the old value and its
// operands.
var atomicFns = map[dxbc.Opcode]func(old, a, b uint32) uint32{
	dxbc.OpAtomicAnd: func(o, a, _ uint32) uint32 { return o & a }, dxbc.OpImmAtomicAnd: func(o, a, _ uint32) uint32 { return o & a },
	dxbc.OpAtomicOr: func(o, a, _ uint32) uint32 { return o | a }, dxbc.OpImmAtomicOr: func(o, a, _ uint32) uint32 { return o | a },
	dxbc.OpAtomicXor: func(o, a, _ uint32) uint32 { return o ^ a }, dxbc.OpImmAtomicXor: func(o, a, _ uint32) uint32 { return o ^ a },
	dxbc.OpAtomicIadd: func(o, a, _ uint32) uint32 { return o + a }, dxbc.OpImmAtomicIadd: func(o, a, _ uint32) uint32 { return o + a },
	dxbc.OpAtomicImax: imaxFn, dxbc.OpImmAtomicImax: imaxFn, dxbc.OpAtomicImin: iminFn, dxbc.OpImmAtomicImin: iminFn,
	dxbc.OpAtomicUmax: func(o, a, _ uint32) uint32 { return max(o, a) }, dxbc.OpImmAtomicUmax: func(o, a, _ uint32) uint32 { return max(o, a) },
	dxbc.OpAtomicUmin: func(o, a, _ uint32) uint32 { return min(o, a) }, dxbc.OpImmAtomicUmin: func(o, a, _ uint32) uint32 { return min(o, a) },
	dxbc.OpImmAtomicExch:  func(_, a, _ uint32) uint32 { return a },
	dxbc.OpAtomicCmpStore: cmpExch, dxbc.OpImmAtomicCmpExch: cmpExch,
}

func imaxFn(o, a, _ uint32) uint32 { return uint32(max(int32(o), int32(a))) }
func iminFn(o, a, _ uint32) uint32 { return uint32(min(int32(o), int32(a))) }

// cmpExch stores b where the old value equals a.
func cmpExch(o, a, b uint32) uint32 {
	if o == a {
		return b
	}
	return o
}

func atomic(t *thread, in *dxbc.Instruction) error {
	f := atomicFns[in.Op]
	ops := in.Operands
	var dest *dxbc.Operand
	if in.Op >= dxbc.OpImmAtomicAlloc {
		dest, ops = &ops[0], ops[1:]
	}
	w, err := t.atomicWord(&ops[0], t.src(&ops[1]))
	if err != nil {
		return err
	}
	a := t.src(&ops[2])[0]
	var b uint32
	if len(ops) > 3 {
		b = t.src(&ops[3])[0]
	}
	old := w.get()
	w.set(f(old, a, b))
	if dest != nil {
		return t.store(dest, Value{old, old, old, old}, false)
	}
	return nil
}

func counter(t *thread, in *dxbc.Instruction) error {
	reg := uint32(t.index(&in.Operands[1], 0))
	if t.w.counters == nil {
		t.w.counters = map[uint32]uint32{}
	}
	n := t.w.counters[reg]
	if in.Op == dxbc.OpImmAtomicAlloc {
		t.w.counters[reg] = n + 1
	} else {
		n--
		t.w.counters[reg] = n
	}
	return t.store(&in.Operands[0], Value{n, n, n, n}, false)
}

// evaluate is an eval_* instruction: the input as interpolated at the
// pixel's center, where it is defined.
func evaluate(t *thread, in *dxbc.Instruction) error {
	return t.store(&in.Operands[0], t.src(&in.Operands[1]), false)
}

func initResources() {
	for _, op := range []dxbc.Opcode{dxbc.OpSample, dxbc.OpSampleB, dxbc.OpSampleL, dxbc.OpSampleD, dxbc.OpSampleC, dxbc.OpSampleCLz} {
		ops[op] = sample
	}
	for _, op := range []dxbc.Opcode{dxbc.OpGather4, dxbc.OpGather4C, dxbc.OpGather4Po, dxbc.OpGather4PoC} {
		ops[op] = gather
	}
	ops[dxbc.OpLd], ops[dxbc.OpLdMs] = ld, ld
	ops[dxbc.OpLdUavTyped], ops[dxbc.OpStoreUavTyped] = ldUAVTyped, storeUAVTyped
	ops[dxbc.OpResinfo], ops[dxbc.OpBufinfo], ops[dxbc.OpLod] = resinfo, bufinfo, lod
	ops[dxbc.OpSampleInfo], ops[dxbc.OpSamplePos] = sampleinfo, samplepos
	ops[dxbc.OpLdStructured], ops[dxbc.OpStoreStructured] = ldStructured, storeStructured
	ops[dxbc.OpLdRaw], ops[dxbc.OpStoreRaw] = ldRaw, storeRaw
	for op := range atomicFns {
		ops[op] = atomic
	}
	ops[dxbc.OpImmAtomicAlloc], ops[dxbc.OpImmAtomicConsume] = counter, counter
	for _, op := range []dxbc.Opcode{dxbc.OpEvalCentroid, dxbc.OpEvalSampleIndex, dxbc.OpEvalSnapped} {
		ops[op] = evaluate
	}
}
