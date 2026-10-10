// Package sim runs DXBC programs on the CPU against simulated resources.
//
// It is the oracle that a decompiled shader, compiled again, computes what
// the original did: both run on the same inputs, constant buffers and
// resources, and their outputs and memory writes are compared. What the
// simulator simplifies it simplifies for both: a derivative is 0, a texture is
// a smooth function of its coordinates and level of detail, a buffer's
// contents a hash of the address. One invocation runs at a time; a compute
// shader's group runs thread after thread, each up to its next barrier.
package sim

import (
	"errors"
	"fmt"

	"github.com/McReaper/t7_dxbc/dxbc"
)

// Value is a register's four components, as bits.
type Value [4]uint32

// ErrSteps is returned when an invocation runs too long: a loop the inputs
// never end.
var ErrSteps = errors.New("step limit reached")

// maxSteps bounds one invocation.
const maxSteps = 1 << 20

// view names a t#, u# or g# register.
type view struct {
	t   dxbc.OperandType
	reg uint32
}

// Machine is a program ready to run: its body and the declarations it reads.
type Machine struct {
	p        *dxbc.Program
	body     []dxbc.Instruction
	flow     flow
	temps    int
	idx      map[uint32]uint32    // x# -> elements
	tgsm     map[uint32][2]uint32 // g# -> stride (0: raw), dwords
	strides  map[view]uint32      // a structured t# or u# -> its stride
	icb      []uint32
	group    [3]uint32
	binds    map[view]*dxbc.Binding
	elems    map[view]*dxbc.Type // a structured view -> its element type
	vertices int                 // a geometry shader's input vertices
}

// New reads a program's declarations.
func New(p *dxbc.Program, ref *dxbc.Reflection) (*Machine, error) {
	m := &Machine{p: p, idx: map[uint32]uint32{}, tgsm: map[uint32][2]uint32{}, strides: map[view]uint32{},
		binds: map[view]*dxbc.Binding{}, elems: map[view]*dxbc.Type{}}
	m.readReflection(ref)
	for i := range p.Instructions {
		in := &p.Instructions[i]
		if in.Op.IsDeclaration() {
			m.declare(in)
			continue
		}
		m.body = append(m.body, *in)
	}
	if err := m.checkLimits(); err != nil {
		return nil, err
	}
	f, err := newFlow(m.body)
	if err != nil {
		return nil, err
	}
	m.flow = f
	return m, nil
}

func (m *Machine) readReflection(ref *dxbc.Reflection) {
	buffers := map[string]*dxbc.Buffer{}
	for i := range ref.Buffers {
		buffers[ref.Buffers[i].Name] = &ref.Buffers[i]
	}
	for i := range ref.Bindings {
		b := &ref.Bindings[i]
		t := dxbc.OperandResource
		switch {
		case b.Input == dxbc.InputCBuffer || b.Input == dxbc.InputSampler:
			continue
		case b.Input.IsUAV():
			t = dxbc.OperandUAV
		}
		for k := range max(b.Count, 1) {
			v := view{t, b.Point + k}
			m.binds[v] = b
			if buf := buffers[b.Name]; buf != nil && buf.Kind == dxbc.ResourceBindInfo && len(buf.Vars) == 1 {
				m.elems[v] = buf.Vars[0].Type
			}
		}
	}
}

func (m *Machine) declare(in *dxbc.Instruction) {
	reg := func() uint32 {
		if len(in.Operands) == 0 || len(in.Operands[0].Index) == 0 {
			return 0
		}
		return uint32(in.Operands[0].Index[0].Imm)
	}
	pay := func(i int) uint32 {
		if i < len(in.Payload) {
			return in.Payload[i]
		}
		return 0
	}
	switch in.Op {
	case dxbc.OpCustomdata:
		if in.CustomdataClass() == dxbc.CustomdataICB {
			m.icb = in.Payload
		}
	case dxbc.OpDclTemps:
		m.temps = int(pay(0))
	case dxbc.OpDclIndexableTemp:
		m.idx[pay(0)] = pay(1)
	case dxbc.OpDclThreadGroupSharedMemoryRaw:
		m.tgsm[reg()] = [2]uint32{0, pay(0) / 4}
	case dxbc.OpDclThreadGroupSharedMemoryStructured:
		m.tgsm[reg()] = [2]uint32{pay(0), pay(0) / 4 * pay(1)}
	case dxbc.OpDclResourceStructured:
		m.strides[view{dxbc.OperandResource, reg()}] = pay(0)
	case dxbc.OpDclUnorderedAccessViewStructured:
		m.strides[view{dxbc.OperandUAV, reg()}] = pay(0)
	case dxbc.OpDclThreadGroup:
		m.group = [3]uint32{pay(0), pay(1), pay(2)}
	case dxbc.OpDclInput, dxbc.OpDclInputSiv, dxbc.OpDclInputSgv:
		if len(in.Operands) > 0 && len(in.Operands[0].Index) == 2 {
			m.vertices = int(in.Operands[0].Index[0].Imm)
		}
	}
}

// flow is where each control-flow instruction goes.
type flow struct {
	jump  map[int]int      // if -> its else or endif, else -> endif, endloop -> loop, break -> past the loop or switch, continue -> loop
	cases map[int][]caseAt // switch -> its labels
	end   map[int]int      // switch -> endswitch
}

type caseAt struct {
	value     uint32
	pc        int
	isDefault bool
}

type frame struct {
	op     dxbc.Opcode
	start  int
	elseAt int
	breaks []int
	conts  []int
}

func newFlow(body []dxbc.Instruction) (flow, error) {
	f := flow{jump: map[int]int{}, cases: map[int][]caseAt{}, end: map[int]int{}}
	var stack []*frame
	top := func(ops ...dxbc.Opcode) *frame {
		for i := len(stack) - 1; i >= 0; i-- {
			for _, op := range ops {
				if stack[i].op == op {
					return stack[i]
				}
			}
		}
		return nil
	}
	for i := range body {
		in := &body[i]
		switch in.Op {
		case dxbc.OpIf, dxbc.OpLoop, dxbc.OpSwitch:
			stack = append(stack, &frame{op: in.Op, start: i, elseAt: -1})
			continue
		case dxbc.OpElse, dxbc.OpEndif, dxbc.OpEndloop, dxbc.OpEndswitch, dxbc.OpCase, dxbc.OpDefault:
			if err := f.close(stack, i, in); err != nil {
				return f, err
			}
			if in.Op == dxbc.OpEndif || in.Op == dxbc.OpEndloop || in.Op == dxbc.OpEndswitch {
				stack = stack[:len(stack)-1]
			}
		case dxbc.OpBreak, dxbc.OpBreakc:
			fr := top(dxbc.OpLoop, dxbc.OpSwitch)
			if fr == nil {
				return f, fmt.Errorf("instruction %d: break outside a loop or switch", i)
			}
			fr.breaks = append(fr.breaks, i)
		case dxbc.OpContinue, dxbc.OpContinuec:
			fr := top(dxbc.OpLoop)
			if fr == nil {
				return f, fmt.Errorf("instruction %d: continue outside a loop", i)
			}
			fr.conts = append(fr.conts, i)
		}
	}
	if len(stack) > 0 {
		return f, fmt.Errorf("%s at instruction %d never closed", stack[len(stack)-1].op.Name(), stack[len(stack)-1].start)
	}
	return f, nil
}

// close records where the block on top of the stack goes when instruction i
// ends or splits it.
func (f *flow) close(stack []*frame, i int, in *dxbc.Instruction) error {
	if len(stack) == 0 {
		return fmt.Errorf("instruction %d: %s outside a block", i, in.Op.Name())
	}
	fr := stack[len(stack)-1]
	want := map[dxbc.Opcode]dxbc.Opcode{dxbc.OpElse: dxbc.OpIf, dxbc.OpEndif: dxbc.OpIf, dxbc.OpEndloop: dxbc.OpLoop,
		dxbc.OpEndswitch: dxbc.OpSwitch, dxbc.OpCase: dxbc.OpSwitch, dxbc.OpDefault: dxbc.OpSwitch}[in.Op]
	if fr.op != want {
		return fmt.Errorf("instruction %d: %s closes a %s", i, in.Op.Name(), fr.op.Name())
	}
	switch in.Op {
	case dxbc.OpElse:
		fr.elseAt = i
		f.jump[fr.start] = i
	case dxbc.OpEndif:
		if fr.elseAt >= 0 {
			f.jump[fr.elseAt] = i
		} else {
			f.jump[fr.start] = i
		}
	case dxbc.OpEndloop:
		f.jump[i] = fr.start
		for _, b := range fr.breaks {
			f.jump[b] = i + 1
		}
		for _, c := range fr.conts {
			f.jump[c] = fr.start
		}
	case dxbc.OpEndswitch:
		f.end[fr.start] = i
		for _, b := range fr.breaks {
			f.jump[b] = i + 1
		}
	case dxbc.OpCase:
		var v uint32
		if len(in.Operands) > 0 && len(in.Operands[0].Immediate) > 0 {
			v = in.Operands[0].Immediate[0]
		}
		f.cases[fr.start] = append(f.cases[fr.start], caseAt{value: v, pc: i})
	case dxbc.OpDefault:
		f.cases[fr.start] = append(f.cases[fr.start], caseAt{pc: i, isDefault: true})
	}
	return nil
}

// Direct3D 11's limits on what a shader declares.
const (
	maxTemps        = 4096
	maxGroupShared  = 32768 / 4 // dwords
	maxGroupThreads = 1024
)

func (m *Machine) checkLimits() error {
	n := uint64(m.temps)
	for _, size := range m.idx {
		n += uint64(size)
	}
	var shared uint64
	for _, d := range m.tgsm {
		shared += uint64(d[1])
	}
	threads := uint64(m.group[0]) * uint64(m.group[1]) * uint64(m.group[2])
	if n > maxTemps || shared > maxGroupShared || threads > maxGroupThreads {
		return fmt.Errorf("declarations past Direct3D's limits: %d temporaries, %d dwords shared, %d threads", n, shared, threads)
	}
	return nil
}
