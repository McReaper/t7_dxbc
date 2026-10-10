package dxbc

import (
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strings"
)

// Decompile writes a shader back as HLSL that recompiles to the same
// interface: the constant buffers with their variables at their offsets, the
// resources and samplers under their names (a structured buffer with its
// element struct), the signatures as the entry point's parameters, and every
// instruction as one or more statements.
//
// DXBC registers have no type; the HLSL keeps them as float4 and reads or
// writes an integer operation's operands through asint, asuint and asfloat,
// which reinterpret bits and cost nothing, so the arithmetic is exact. The
// names are the compiler's own, from the reflection; the temporaries are r#,
// the inputs v# and the outputs o#, as in the disassembly.
func Decompile(b []byte) (string, error) {
	c, err := Parse(b)
	if err != nil {
		return "", err
	}
	code := c.Chunk("SHEX", "SHDR")
	if code == nil {
		return "", errors.New("no SHEX or SHDR chunk: not a shader")
	}
	p, err := DecodeProgram(code)
	if err != nil {
		return "", err
	}
	ref := &Reflection{}
	if rdef := c.Chunk("RDEF"); rdef != nil {
		if ref, err = ReadReflection(rdef); err != nil {
			return "", err
		}
	}
	in, out, err := c.Signatures()
	if err != nil {
		return "", err
	}
	switch p.Stage {
	case StagePixel, StageVertex, StageCompute, StageGeometry:
	default:
		return "", fmt.Errorf("stage %d: hull and domain shaders aren't decompiled", p.Stage)
	}
	e := newEmitter(p, ref, in, out)
	return e.emit()
}

// emitter writes one shader.
type emitter struct {
	p        *Program
	ref      *Reflection
	in, out  []Element
	sb       strings.Builder
	depth    int
	cbuffers map[uint32]*cbufferLayout // by register
	structs  map[string]*Type          // a structured buffer's name -> its element type
	icb      []uint32
	temps    int
	idxTemps map[uint32][2]uint32 // x# -> size, components
	tgsm     map[uint32][2]uint32 // g# -> stride (0: raw), count
	group    [3]uint32
	inMods   map[uint32]string // v# -> interpolation modifiers
	helpers  map[string]bool   // the helper functions the statements call
	current  *Instruction      // the instruction being written
	intRegs  map[uint32]uint8  // r# -> the components that hold integers, in i#
	intX     map[uint32]bool   // x# -> an array of uint4
	dry      bool              // finding the banks: write nothing
	gs       geometry
	uavFlags map[uint32]uint32 // u# -> its declaration's flags
	global   uint32            // the global flags

	inDecls, outDecls   []sigDecl       // how the signatures are declared
	inFields, outFields map[slot]string // what names each element in them

	generated      map[string]string // helpers written for this shader, by name
	generatedOrder []string
	unknown        map[string]bool
}

func newEmitter(p *Program, ref *Reflection, in, out []Element) *emitter {
	e := &emitter{p: p, ref: ref, in: in, out: out, cbuffers: map[uint32]*cbufferLayout{},
		structs: map[string]*Type{}, idxTemps: map[uint32][2]uint32{}, tgsm: map[uint32][2]uint32{},
		inMods: map[uint32]string{}, helpers: map[string]bool{}, unknown: map[string]bool{}, uavFlags: map[uint32]uint32{},
		intRegs: map[uint32]uint8{}, intX: map[uint32]bool{}, generated: map[string]string{}}
	buffers := map[string]*Buffer{}
	for i := range ref.Buffers {
		buffers[ref.Buffers[i].Name] = &ref.Buffers[i]
	}
	for i := range ref.Bindings {
		bd := &ref.Bindings[i]
		switch bd.Input {
		case InputCBuffer:
			if buf := buffers[bd.Name]; buf != nil && buf.Kind == CBuffer {
				e.cbuffers[bd.Point] = newCBufferLayout(buf)
			}
		default:
			if buf := buffers[bd.Name]; buf != nil && buf.Kind == ResourceBindInfo && len(buf.Vars) == 1 {
				e.structs[bd.Name] = buf.Vars[0].Type
			}
		}
	}
	return e
}

func (e *emitter) line(format string, args ...any) {
	if e.dry {
		return
	}
	e.sb.WriteString(strings.Repeat("  ", e.depth))
	fmt.Fprintf(&e.sb, format, args...)
	e.sb.WriteByte('\n')
}

func (e *emitter) emit() (string, error) {
	var body []Instruction
	for _, in := range e.p.Instructions {
		if !e.declare(in) {
			body = append(body, in)
		}
	}
	if err := e.checkLimits(); err != nil {
		return "", err
	}
	if e.p.Stage == StageGeometry {
		if err := e.checkGeometry(); err != nil {
			return "", err
		}
	}
	e.layoutSignatures()
	e.findBanks(body)
	e.declarations()
	e.signatureStructs()
	head := e.sb.String()
	e.sb.Reset()
	e.entry()
	e.depth++
	e.locals()
	if err := e.statements(body); err != nil {
		return "", err
	}
	e.depth--
	e.line("}")
	src := head + e.helperDecls() + e.sb.String()
	if len(e.unknown) > 0 {
		ops := make([]string, 0, len(e.unknown))
		for k := range e.unknown {
			ops = append(ops, k)
		}
		sort.Strings(ops)
		return src, fmt.Errorf("not decompiled: %s", strings.Join(ops, ", "))
	}
	return src, nil
}

// Direct3D 11's limits on what a shader declares.
const (
	maxTemps        = 4096  // temporaries, indexable ones included
	maxGroupShared  = 32768 // bytes of group-shared memory
	maxGroupThreads = 1024
)

// checkLimits rejects declarations no valid shader makes, which would only
// make the HLSL huge.
func (e *emitter) checkLimits() error {
	n := uint64(e.temps)
	for _, d := range e.idxTemps {
		n += uint64(d[0])
	}
	if n > maxTemps {
		return fmt.Errorf("%d temporaries declared, more than Direct3D allows", n)
	}
	var shared uint64
	for _, d := range e.tgsm {
		if d[0] == 0 {
			shared += uint64(d[1])
		} else {
			shared += uint64(d[0]) * uint64(d[1])
		}
	}
	if shared > maxGroupShared {
		return fmt.Errorf("%d bytes of group-shared memory declared, more than Direct3D allows", shared)
	}
	if t := uint64(e.group[0]) * uint64(e.group[1]) * uint64(e.group[2]); t > maxGroupThreads {
		return fmt.Errorf("%d threads per group declared, more than Direct3D allows", t)
	}
	return nil
}

// findBanks runs the statements without writing them until the temporaries
// whose components receive integers are known: a mov's type is its source's,
// so one pass can reveal more.
func (e *emitter) findBanks(body []Instruction) {
	e.dry = true
	for {
		n := e.intCount()
		_ = e.statements(body)
		if e.intCount() == n {
			break
		}
	}
	e.dry = false
	e.depth = 0
	e.unknown = map[string]bool{}
	e.helpers = map[string]bool{}
	e.generated, e.generatedOrder = map[string]string{}, nil
}

// intCount sums the integer components found so far.
func (e *emitter) intCount() int {
	n := len(e.intX)
	for _, m := range e.intRegs {
		n += bits.OnesCount8(m)
	}
	return n
}

// helperDecls writes the helper functions the statements called.
func (e *emitter) helperDecls() string {
	names := make([]string, 0, len(e.helpers))
	for n := range e.helpers {
		names = append(names, n)
	}
	sort.Strings(names)
	var sb strings.Builder
	for _, n := range names {
		sb.WriteString(helperSource(n))
	}
	for _, n := range e.generatedOrder {
		sb.WriteString(e.generated[n])
	}
	if sb.Len() > 0 {
		sb.WriteByte('\n')
	}
	return sb.String()
}

// declare records what a declaration says, and reports whether it is one.
func (e *emitter) declare(in Instruction) bool {
	if f, ok := declarations[in.Op]; ok {
		f(e, &in)
		return true
	}
	return in.Op.IsDeclaration()
}

// operandRegister is a declaration's register number, or false.
func operandRegister(in *Instruction) (uint32, bool) {
	if len(in.Operands) == 0 || len(in.Operands[0].Index) == 0 {
		return 0, false
	}
	return uint32(in.Operands[0].Index[0].Imm), true
}

// declarations are the declarations the HLSL needs.
var declarations = map[Opcode]func(e *emitter, in *Instruction){
	OpCustomdata: func(e *emitter, in *Instruction) {
		if in.CustomdataClass() == CustomdataICB {
			e.icb = in.Payload
		}
	},
	OpDclTemps: func(e *emitter, in *Instruction) {
		if len(in.Payload) > 0 {
			e.temps = int(in.Payload[0])
		}
	},
	OpDclIndexableTemp: func(e *emitter, in *Instruction) {
		if len(in.Payload) >= 3 {
			e.idxTemps[in.Payload[0]] = [2]uint32{in.Payload[1], in.Payload[2]}
		}
	},
	OpDclThreadGroupSharedMemoryRaw: func(e *emitter, in *Instruction) {
		if g, ok := operandRegister(in); ok && len(in.Payload) > 0 {
			e.tgsm[g] = [2]uint32{0, in.Payload[0]}
		}
	},
	OpDclThreadGroupSharedMemoryStructured: func(e *emitter, in *Instruction) {
		if g, ok := operandRegister(in); ok && len(in.Payload) > 1 {
			e.tgsm[g] = [2]uint32{in.Payload[0], in.Payload[1]}
		}
	},
	OpDclThreadGroup: func(e *emitter, in *Instruction) {
		if len(in.Payload) >= 3 {
			copy(e.group[:], in.Payload)
		}
	},
	OpDclInputPs:                       declareInputPs,
	OpDclGlobalFlags:                   func(e *emitter, in *Instruction) { e.global = in.GlobalFlags() },
	OpDclUnorderedAccessViewTyped:      declareUAV,
	OpDclUnorderedAccessViewRaw:        declareUAV,
	OpDclUnorderedAccessViewStructured: declareUAV,
	OpDclInputPsSiv:                    declareInputPs,
	OpDclInputPsSgv:                    declareInputPs,
}

func declareUAV(e *emitter, in *Instruction) {
	if u, ok := operandRegister(in); ok {
		e.uavFlags[u] = in.UAVFlags()
	}
}

func declareInputPs(e *emitter, in *Instruction) {
	if v, ok := operandRegister(in); ok {
		e.inMods[v] = interpolationModifiers[in.InterpolationMode()]
	}
}

// interpolationModifiers spell a pixel shader input's interpolation in HLSL.
var interpolationModifiers = map[uint32]string{
	1: "nointerpolation ", 3: "centroid ", 4: "noperspective ", 5: "noperspective centroid ", 6: "sample ",
	7: "noperspective sample ",
}
