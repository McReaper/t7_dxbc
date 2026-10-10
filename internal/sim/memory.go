package sim

import (
	"fmt"

	"github.com/McReaper/t7dxbc"
)

// word is one dword of a buffer or of group-shared memory, read and written.
type word struct {
	get func() uint32
	set func(uint32)
}

// word finds the dword a structured or raw access reaches: element index
// and byte offset for a structured view or group memory, a byte address for a
// raw one. Typed UAVs are reached by coordinates instead.
func (t *thread) word(op *t7dxbc.Operand, index, byteOff uint32) (word, error) {
	reg := uint32(t.index(op, 0))
	if op.Type == t7dxbc.OperandTGSM {
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
	if op.Type != t7dxbc.OperandUAV {
		return word{mock, func(uint32) {}}, nil
	}
	if b.Input == t7dxbc.InputUAVTyped {
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
func memberIsFloat(t *t7dxbc.Type, off uint32) bool {
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
	if t.Class == t7dxbc.ClassStruct {
		for i := len(t.Members) - 1; i >= 0; i-- {
			if off >= t.Members[i].Offset {
				return memberIsFloat(t.Members[i].Type, off-t.Members[i].Offset)
			}
		}
		return false
	}
	return t.Base == t7dxbc.TypeFloat
}

// tightSize is a type's size in a structured buffer.
func tightSize(t *t7dxbc.Type) uint32 {
	var one uint32
	switch t.Class {
	case t7dxbc.ClassScalar:
		one = 4
	case t7dxbc.ClassVector:
		one = 4 * uint32(t.Cols)
	case t7dxbc.ClassMatrixRows, t7dxbc.ClassMatrixColumns:
		one = 4 * uint32(t.Rows) * uint32(t.Cols)
	case t7dxbc.ClassStruct:
		for _, m := range t.Members {
			one = max(one, m.Offset+tightSize(m.Type))
		}
	}
	return one * uint32(max(t.Elements, 1))
}

func ldStructured(t *thread, in *t7dxbc.Instruction) error {
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

func ldRaw(t *thread, in *t7dxbc.Instruction) error {
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

func storeStructured(t *thread, in *t7dxbc.Instruction) error {
	dst := &in.Operands[0]
	idx, off, v := t.src(&in.Operands[1])[0], t.src(&in.Operands[2])[0], t.src(&in.Operands[3])
	return t.storeWords(dst, v, func(c uint32) (word, error) { return t.word(dst, idx, off+4*c) })
}

func storeRaw(t *thread, in *t7dxbc.Instruction) error {
	dst := &in.Operands[0]
	addr, v := t.src(&in.Operands[1])[0], t.src(&in.Operands[2])
	return t.storeWords(dst, v, func(c uint32) (word, error) { return t.word(dst, 0, addr+4*c) })
}

// storeWords writes the destination mask's components of v, each to its
// dword.
func (t *thread) storeWords(dst *t7dxbc.Operand, v Value, at func(c uint32) (word, error)) error {
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
func (t *thread) atomicWord(op *t7dxbc.Operand, addr Value) (word, error) {
	if op.Type == t7dxbc.OperandUAV {
		b, err := t.binding(op)
		if err != nil {
			return word{}, err
		}
		if b.Input == t7dxbc.InputUAVTyped {
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

func (t *thread) isStructured(op *t7dxbc.Operand) bool {
	if op.Type == t7dxbc.OperandTGSM {
		return t.m.tgsm[uint32(t.index(op, 0))][0] != 0
	}
	_, ok := t.m.strides[view{op.Type, uint32(t.index(op, 0))}]
	return ok
}

// atomicFns are what each atomic operation makes of the old value and its
// operands.
var atomicFns = map[t7dxbc.Opcode]func(old, a, b uint32) uint32{
	t7dxbc.OpAtomicAnd: func(o, a, _ uint32) uint32 { return o & a }, t7dxbc.OpImmAtomicAnd: func(o, a, _ uint32) uint32 { return o & a },
	t7dxbc.OpAtomicOr: func(o, a, _ uint32) uint32 { return o | a }, t7dxbc.OpImmAtomicOr: func(o, a, _ uint32) uint32 { return o | a },
	t7dxbc.OpAtomicXor: func(o, a, _ uint32) uint32 { return o ^ a }, t7dxbc.OpImmAtomicXor: func(o, a, _ uint32) uint32 { return o ^ a },
	t7dxbc.OpAtomicIadd: func(o, a, _ uint32) uint32 { return o + a }, t7dxbc.OpImmAtomicIadd: func(o, a, _ uint32) uint32 { return o + a },
	t7dxbc.OpAtomicImax: imaxFn, t7dxbc.OpImmAtomicImax: imaxFn, t7dxbc.OpAtomicImin: iminFn, t7dxbc.OpImmAtomicImin: iminFn,
	t7dxbc.OpAtomicUmax: func(o, a, _ uint32) uint32 { return max(o, a) }, t7dxbc.OpImmAtomicUmax: func(o, a, _ uint32) uint32 { return max(o, a) },
	t7dxbc.OpAtomicUmin: func(o, a, _ uint32) uint32 { return min(o, a) }, t7dxbc.OpImmAtomicUmin: func(o, a, _ uint32) uint32 { return min(o, a) },
	t7dxbc.OpImmAtomicExch:  func(_, a, _ uint32) uint32 { return a },
	t7dxbc.OpAtomicCmpStore: cmpExch, t7dxbc.OpImmAtomicCmpExch: cmpExch,
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

func atomic(t *thread, in *t7dxbc.Instruction) error {
	f := atomicFns[in.Op]
	ops := in.Operands
	var dest *t7dxbc.Operand
	if in.Op >= t7dxbc.OpImmAtomicAlloc {
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

func counter(t *thread, in *t7dxbc.Instruction) error {
	reg := uint32(t.index(&in.Operands[1], 0))
	if t.w.counters == nil {
		t.w.counters = map[uint32]uint32{}
	}
	n := t.w.counters[reg]
	if in.Op == t7dxbc.OpImmAtomicAlloc {
		t.w.counters[reg] = n + 1
	} else {
		n--
		t.w.counters[reg] = n
	}
	return t.store(&in.Operands[0], Value{n, n, n, n}, false)
}

// evaluate is an eval_* instruction: the input as interpolated at the
// pixel's center, where it is defined.
func evaluate(t *thread, in *t7dxbc.Instruction) error {
	return t.store(&in.Operands[0], t.src(&in.Operands[1]), false)
}

func initResources() {
	for _, op := range []t7dxbc.Opcode{t7dxbc.OpSample, t7dxbc.OpSampleB, t7dxbc.OpSampleL, t7dxbc.OpSampleD, t7dxbc.OpSampleC, t7dxbc.OpSampleCLz} {
		ops[op] = sample
	}
	for _, op := range []t7dxbc.Opcode{t7dxbc.OpGather4, t7dxbc.OpGather4C, t7dxbc.OpGather4Po, t7dxbc.OpGather4PoC} {
		ops[op] = gather
	}
	ops[t7dxbc.OpLd], ops[t7dxbc.OpLdMs] = ld, ld
	ops[t7dxbc.OpLdUavTyped], ops[t7dxbc.OpStoreUavTyped] = ldUAVTyped, storeUAVTyped
	ops[t7dxbc.OpResinfo], ops[t7dxbc.OpBufinfo], ops[t7dxbc.OpLod] = resinfo, bufinfo, lod
	ops[t7dxbc.OpSampleInfo], ops[t7dxbc.OpSamplePos] = sampleinfo, samplepos
	ops[t7dxbc.OpLdStructured], ops[t7dxbc.OpStoreStructured] = ldStructured, storeStructured
	ops[t7dxbc.OpLdRaw], ops[t7dxbc.OpStoreRaw] = ldRaw, storeRaw
	for op := range atomicFns {
		ops[op] = atomic
	}
	ops[t7dxbc.OpImmAtomicAlloc], ops[t7dxbc.OpImmAtomicConsume] = counter, counter
	for _, op := range []t7dxbc.Opcode{t7dxbc.OpEvalCentroid, t7dxbc.OpEvalSampleIndex, t7dxbc.OpEvalSnapped} {
		ops[op] = evaluate
	}
}
