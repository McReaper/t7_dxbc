package sim

import (
	"fmt"
	"maps"

	"github.com/McReaper/t7dxbc/dxbc"
)

// World is what a run's invocations share: the constant buffers, the
// unordered-access memory they write, the seed of every simulated resource.
type World struct {
	CB       map[uint32][]uint32 // cb# -> its dwords
	Seed     uint32
	UAV      map[Cell]uint32 // the cells written
	counters map[uint32]uint32
	counting bool // integers in the float variables
}

// Cell is one dword of unordered-access memory: u#, its address (coordinates,
// an element index and dword, or a dword index), and a component.
type Cell struct {
	Reg, A, B, C, Comp uint32
}

// Clone copies a world, so two programs can start from the same one.
func (w *World) Clone() *World {
	c := *w
	c.UAV = maps.Clone(w.UAV)
	c.counters = maps.Clone(w.counters)
	if c.UAV == nil {
		c.UAV = map[Cell]uint32{}
	}
	if c.counters == nil {
		c.counters = map[uint32]uint32{}
	}
	return &c
}

// Result is what one invocation leaves.
type Result struct {
	Outputs   map[uint32]Value            // o#
	System    map[dxbc.OperandType]uint32 // oDepth, oMask…
	Discarded bool
	Steps     int
	// Emitted are the vertices a geometry shader appended, its outputs when
	// it emitted each; a nil one is a cut, the end of a strip
	Emitted []map[uint32]Value
}

// VertexInput is the key of a geometry shader's input register for one of
// its primitive's vertices, v[vertex][reg].
func VertexInput(vertex, reg uint32) uint32 { return vertex<<16 | reg }

// InputVertices is how many vertices a geometry shader's primitive has.
func (m *Machine) InputVertices() int { return m.vertices }

// thread is one invocation's state.
type thread struct {
	m      *Machine
	w      *World
	r      []Value
	x      map[uint32][]Value
	in     map[uint32]Value
	sys    map[dxbc.OperandType]Value
	g      map[uint32][]uint32 // the group's shared memory
	res    Result
	pc     int
	done   bool
	waits  bool // stopped at a barrier
	failed error
}

func (m *Machine) thread(w *World, in map[uint32]Value, sys map[dxbc.OperandType]Value, g map[uint32][]uint32) *thread {
	t := &thread{m: m, w: w, r: make([]Value, m.temps), x: map[uint32][]Value{}, in: in, sys: sys, g: g,
		res: Result{Outputs: map[uint32]Value{}, System: map[dxbc.OperandType]uint32{}}}
	for x, n := range m.idx {
		t.x[x] = make([]Value, n)
	}
	return t
}

// Run runs one invocation of a pixel or vertex shader: inputs by v# register,
// system values by operand type.
func (m *Machine) Run(w *World, in map[uint32]Value, sys map[dxbc.OperandType]Value) (Result, error) {
	t := m.thread(w, in, sys, m.sharedMemory())
	for !t.done && t.failed == nil {
		t.step()
	}
	return t.res, t.failed
}

func (m *Machine) sharedMemory() map[uint32][]uint32 {
	g := map[uint32][]uint32{}
	for reg, d := range m.tgsm {
		g[reg] = make([]uint32, d[1])
	}
	return g
}

// RunGroup runs one thread group of a compute shader. Each thread runs to its
// next barrier before the next one starts, so what a barrier orders holds.
func (m *Machine) RunGroup(w *World, group [3]uint32) error {
	g := m.sharedMemory()
	var threads []*thread
	for z := range m.group[2] {
		for y := range m.group[1] {
			for x := range m.group[0] {
				id := Value{x, y, z}
				flat := (z*m.group[1]+y)*m.group[0] + x
				sys := map[dxbc.OperandType]Value{
					dxbc.OperandThreadIDInGroup:          id,
					dxbc.OperandThreadGroupID:            {group[0], group[1], group[2]},
					dxbc.OperandThreadID:                 {group[0]*m.group[0] + x, group[1]*m.group[1] + y, group[2]*m.group[2] + z},
					dxbc.OperandThreadIDInGroupFlattened: {flat, flat, flat, flat},
				}
				threads = append(threads, m.thread(w, nil, sys, g))
			}
		}
	}
	for {
		running := false
		for _, t := range threads {
			if t.done {
				continue
			}
			running = true
			t.waits = false
			for !t.done && !t.waits && t.failed == nil {
				t.step()
			}
			if t.failed != nil {
				return t.failed
			}
		}
		if !running {
			return nil
		}
	}
}

// step runs one instruction.
func (t *thread) step() {
	if t.pc >= len(t.m.body) {
		t.done = true
		return
	}
	if t.res.Steps++; t.res.Steps > maxSteps {
		t.failed = ErrSteps
		return
	}
	in := &t.m.body[t.pc]
	next := t.pc + 1
	if f, ok := flowOps[in.Op]; ok {
		next = f(t, in, t.pc)
	} else if f, ok := ops[in.Op]; ok {
		if err := f(t, in); err != nil {
			t.failed = fmt.Errorf("instruction %d (%s): %w", t.pc, in.Op.Name(), err)
			return
		}
	} else {
		t.failed = fmt.Errorf("instruction %d: %s isn't simulated", t.pc, in.Op.Name())
		return
	}
	t.pc = next
}

// test is a conditional instruction's outcome: _nz any bit set, _z none.
func (t *thread) test(in *dxbc.Instruction) bool {
	v := t.src(&in.Operands[0])[0]
	if in.TestsNonZero() {
		return v != 0
	}
	return v == 0
}

// flowOps move the program counter: each returns the next instruction.
var flowOps = map[dxbc.Opcode]func(t *thread, in *dxbc.Instruction, pc int) int{
	dxbc.OpIf: func(t *thread, in *dxbc.Instruction, pc int) int {
		if t.test(in) {
			return pc + 1
		}
		return t.m.flow.jump[pc] + 1
	},
	dxbc.OpElse:      func(t *thread, _ *dxbc.Instruction, pc int) int { return t.m.flow.jump[pc] + 1 },
	dxbc.OpEndif:     next,
	dxbc.OpLoop:      next,
	dxbc.OpEndloop:   func(t *thread, _ *dxbc.Instruction, pc int) int { return t.m.flow.jump[pc] + 1 },
	dxbc.OpBreak:     func(t *thread, _ *dxbc.Instruction, pc int) int { return t.m.flow.jump[pc] },
	dxbc.OpContinue:  func(t *thread, _ *dxbc.Instruction, pc int) int { return t.m.flow.jump[pc] + 1 },
	dxbc.OpBreakc:    func(t *thread, in *dxbc.Instruction, pc int) int { return t.when(in, pc, t.m.flow.jump[pc]) },
	dxbc.OpContinuec: func(t *thread, in *dxbc.Instruction, pc int) int { return t.when(in, pc, t.m.flow.jump[pc]+1) },
	dxbc.OpSwitch:    switchTo,
	dxbc.OpCase:      next,
	dxbc.OpDefault:   next,
	dxbc.OpEndswitch: next,
	dxbc.OpRet: func(t *thread, _ *dxbc.Instruction, pc int) int {
		t.done = true
		return pc
	},
	dxbc.OpRetc: func(t *thread, in *dxbc.Instruction, pc int) int {
		t.done = t.test(in)
		return pc + 1
	},
	dxbc.OpDiscard: func(t *thread, in *dxbc.Instruction, pc int) int {
		if t.test(in) {
			t.res.Discarded, t.done = true, true
		}
		return pc + 1
	},
	dxbc.OpSync: func(t *thread, _ *dxbc.Instruction, pc int) int {
		t.waits = true
		return pc + 1
	},
}

func next(_ *thread, _ *dxbc.Instruction, pc int) int { return pc + 1 }

// when goes to target if the instruction's test holds.
func (t *thread) when(in *dxbc.Instruction, pc, target int) int {
	if t.test(in) {
		return target
	}
	return pc + 1
}

// switchTo goes to the label matching the selector, the default, or past the
// switch.
func switchTo(t *thread, in *dxbc.Instruction, pc int) int {
	v := t.src(&in.Operands[0])[0]
	target := t.m.flow.end[pc]
	for _, c := range t.m.flow.cases[pc] {
		if c.isDefault {
			target = c.pc
		}
	}
	for _, c := range t.m.flow.cases[pc] {
		if !c.isDefault && c.value == v {
			return c.pc + 1
		}
	}
	return target + 1
}

func emit(t *thread, in *dxbc.Instruction) error {
	if in.Op != dxbc.OpCut && in.Op != dxbc.OpCutStream {
		v := make(map[uint32]Value, len(t.res.Outputs))
		for k, o := range t.res.Outputs {
			v[k] = o
		}
		t.res.Emitted = append(t.res.Emitted, v)
	}
	if in.Op != dxbc.OpEmit && in.Op != dxbc.OpEmitStream {
		t.res.Emitted = append(t.res.Emitted, nil)
	}
	return nil
}

func init() {
	for _, op := range []dxbc.Opcode{dxbc.OpEmit, dxbc.OpEmitStream, dxbc.OpCut, dxbc.OpCutStream, dxbc.OpEmitthencut,
		dxbc.OpEmitthencutStream} {
		ops[op] = emit
	}
}
