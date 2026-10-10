package hlsl

import (
	"fmt"
	"strings"

	"github.com/McReaper/t7_dxbc/dxbc"
)

// atomicIntrinsics name each atomic operation's HLSL intrinsic, and the type
// its value is: signed for imax and imin.
var atomicIntrinsics = map[dxbc.Opcode]struct {
	name string
	t    valType
}{
	dxbc.OpAtomicAnd: {"InterlockedAnd", tUint}, dxbc.OpAtomicOr: {"InterlockedOr", tUint}, dxbc.OpAtomicXor: {"InterlockedXor", tUint},
	dxbc.OpAtomicIadd: {"InterlockedAdd", tUint}, dxbc.OpAtomicImax: {"InterlockedMax", tInt}, dxbc.OpAtomicImin: {"InterlockedMin", tInt},
	dxbc.OpAtomicUmax: {"InterlockedMax", tUint}, dxbc.OpAtomicUmin: {"InterlockedMin", tUint},
	dxbc.OpAtomicCmpStore: {"InterlockedCompareStore", tUint},
	dxbc.OpImmAtomicAnd:   {"InterlockedAnd", tUint}, dxbc.OpImmAtomicOr: {"InterlockedOr", tUint},
	dxbc.OpImmAtomicXor: {"InterlockedXor", tUint}, dxbc.OpImmAtomicIadd: {"InterlockedAdd", tUint},
	dxbc.OpImmAtomicImax: {"InterlockedMax", tInt}, dxbc.OpImmAtomicImin: {"InterlockedMin", tInt},
	dxbc.OpImmAtomicUmax: {"InterlockedMax", tUint}, dxbc.OpImmAtomicUmin: {"InterlockedMin", tUint},
	dxbc.OpImmAtomicExch: {"InterlockedExchange", tUint}, dxbc.OpImmAtomicCmpExch: {"InterlockedCompareExchange", tUint},
}

// atomicStatement writes an atomic operation: atomic_* target, address,
// values; imm_atomic_* dest, target, address, values, which also returns the
// value before.
func (e *emitter) atomicStatement(in *dxbc.Instruction) {
	a := atomicIntrinsics[in.Op]
	ops := in.Operands
	var dest *dxbc.Operand
	if strings.HasPrefix(in.Op.Name(), "imm_") {
		dest, ops = &ops[0], ops[1:]
	}
	if e.dynamicAtomic(a.name, dest, ops) {
		return
	}
	target, method, ok := e.atomicTarget(&ops[0], &ops[1])
	if !ok {
		return
	}
	var args []string
	if method != "" {
		args = append(args, method)
	} else {
		args = append(args, target)
	}
	for i := 2; i < len(ops); i++ {
		args = append(args, e.read(&ops[i], picks(&ops[i], 1), a.t))
	}
	call := a.name
	if method != "" {
		call = target + "." + a.name
	}
	if dest == nil {
		e.line("%s(%s);", call, strings.Join(args, ", "))
		return
	}
	e.line("{")
	e.depth++
	e.line("%s dxbc_before;", a.t)
	e.line("%s(%s, dxbc_before);", call, strings.Join(args, ", "))
	e.write(dest, "dxbc_before", a.t, false)
	e.depth--
	e.line("}")
}

// atomicTarget is what an atomic operation changes: an element expression
// (typed, structured, group memory), or a raw buffer and the byte address
// its method takes.
func (e *emitter) atomicTarget(op, addr *dxbc.Operand) (target, method string, ok bool) {
	if op.Type == dxbc.OperandTGSM {
		g := uint32(op.Index[0].Imm)
		if e.tgsm[g][0] == 0 {
			return fmt.Sprintf("g%d[%s >> 2]", g, e.read(addr, picks(addr, 1), tUint)), "", true
		}
		idx := e.read(addr, first(addr, 1), tUint)
		off := e.read(addr, first(addr, 2)[1:], tUint)
		return fmt.Sprintf("g%d[%s][%s >> 2]", g, idx, off), "", true
	}
	res, b := e.resourceName(op)
	if b == nil {
		return "", "", false
	}
	switch b.Input {
	case dxbc.InputUAVByteAddress:
		return res, e.read(addr, picks(addr, 1), tUint), true
	case dxbc.InputUAVTyped:
		return fmt.Sprintf("%s[%s]", res, e.read(addr, first(addr, uavCoords(b.Dim)), tUint)), "", true
	}
	off, isImm := immediateComponent(addr, 1)
	if !isImm {
		e.unknown["atomic on a structured buffer at a computed offset"] = true
		return "", "", false
	}
	s, found := e.structuredElement(op, e.read(addr, first(addr, 1), tUint), off)
	if !found {
		e.unknown["atomic outside a structured element"] = true
		return "", "", false
	}
	return s.expr, "", true
}

// immediateComponent is component i of an address the instruction reads, when
// that component is a literal.
func immediateComponent(op *dxbc.Operand, i int) (uint32, bool) {
	if op.Type != dxbc.OperandImmediate32 {
		return 0, false
	}
	c := i
	if op.Selection == dxbc.SelectSwizzle {
		c = int(op.Swizzle[i])
	}
	if c >= len(op.Immediate) {
		return 0, false
	}
	return op.Immediate[c], true
}

// counterStatement writes imm_atomic_alloc and imm_atomic_consume: a
// structured buffer's counter.
func (e *emitter) counterStatement(in *dxbc.Instruction) {
	res, b := e.resourceName(&in.Operands[1])
	if b == nil {
		return
	}
	method := "IncrementCounter"
	if in.Op == dxbc.OpImmAtomicConsume {
		method = "DecrementCounter"
	}
	e.write(&in.Operands[0], fmt.Sprintf("%s.%s()", res, method), tUint, false)
}

// syncStatement writes sync as the barrier intrinsic that waits for the same
// memory and threads.
func (e *emitter) syncStatement(in *dxbc.Instruction) {
	f := in.SyncFlags()
	uav, group, threads := f&(8|4) != 0, f&2 != 0, f&1 != 0
	name := "GroupMemoryBarrier"
	switch {
	case uav && group:
		name = "AllMemoryBarrier"
	case uav:
		name = "DeviceMemoryBarrier"
	}
	if threads {
		name += "WithGroupSync"
	}
	e.line("%s();", name)
}

// initResourceStatements registers the resource, memory and barrier
// instructions.
func initResourceStatements() {
	for op := range sampleForms {
		otherStatements[op] = (*emitter).sampleStatement
	}
	for _, op := range []dxbc.Opcode{dxbc.OpGather4, dxbc.OpGather4C, dxbc.OpGather4Po, dxbc.OpGather4PoC} {
		otherStatements[op] = (*emitter).gatherStatement
	}
	for _, op := range []dxbc.Opcode{dxbc.OpLd, dxbc.OpLdMs, dxbc.OpLdUavTyped} {
		otherStatements[op] = (*emitter).ldStatement
	}
	otherStatements[dxbc.OpResinfo] = (*emitter).resinfoStatement
	otherStatements[dxbc.OpLod] = (*emitter).lodStatement
	otherStatements[dxbc.OpSampleInfo] = (*emitter).sampleinfoStatement
	otherStatements[dxbc.OpSamplePos] = (*emitter).sampleposStatement
	for op := range evalFunctions {
		otherStatements[op] = (*emitter).evalStatement
	}
	otherStatements[dxbc.OpLdStructured] = (*emitter).ldStructuredStatement
	otherStatements[dxbc.OpLdRaw] = (*emitter).ldRawStatement
	noSat := map[dxbc.Opcode]func(*emitter, *dxbc.Instruction){
		dxbc.OpStoreUavTyped: (*emitter).storeTypedStatement, dxbc.OpStoreStructured: (*emitter).storeStructuredStatement,
		dxbc.OpStoreRaw: (*emitter).storeRawStatement, dxbc.OpBufinfo: (*emitter).bufinfoStatement,
		dxbc.OpImmAtomicAlloc: (*emitter).counterStatement, dxbc.OpImmAtomicConsume: (*emitter).counterStatement,
		dxbc.OpSync: (*emitter).syncStatement, dxbc.OpNop: func(*emitter, *dxbc.Instruction) {},
	}
	for op := range atomicIntrinsics {
		noSat[op] = (*emitter).atomicStatement
	}
	for op, f := range noSat {
		otherStatements[op] = func(e *emitter, in *dxbc.Instruction, _ bool) { f(e, in) }
	}
}

// dynamicAtomic writes an atomic operation on a structured element at a
// computed offset through a helper, and reports whether it was one.
func (e *emitter) dynamicAtomic(intrinsic string, dest *dxbc.Operand, ops []dxbc.Operand) bool {
	op, addr := &ops[0], &ops[1]
	if op.Type != dxbc.OperandUAV {
		return false
	}
	if _, isImm := immediateComponent(addr, 1); isImm {
		return false
	}
	res, b := e.resourceName(op)
	if b == nil || b.Input == dxbc.InputUAVByteAddress || b.Input == dxbc.InputUAVTyped {
		return false
	}
	t := e.structs[b.Name]
	if t == nil {
		e.unknown["atomic at a computed offset into "+res+", whose element is unknown"] = true
		return true
	}
	acc := dynamicAccess{name: intrinsic, params: []string{"dxbc_a"}}
	if len(ops) > 3 {
		acc.params = append(acc.params, "dxbc_b")
	}
	args := []string{e.read(addr, first(addr, 1), tUint), e.read(addr, first(addr, 2)[1:], tUint)}
	for i := 2; i < len(ops); i++ {
		args = append(args, e.read(&ops[i], picks(&ops[i], 1), tUint))
	}
	call := fmt.Sprintf("%s(%s)", e.dynamicHelper(acc, res, t), strings.Join(args, ", "))
	if dest == nil {
		e.line("%s;", call)
		return true
	}
	e.write(dest, call, tUint, false)
	return true
}
