package dxbc

import (
	"errors"
	"fmt"
)

// Stage is the program type a shader's SHDR/SHEX version token names.
type Stage uint32

// The pipeline stages.
const (
	StagePixel Stage = iota
	StageVertex
	StageGeometry
	StageHull
	StageDomain
	StageCompute
)

// Program is a decoded SHDR or SHEX chunk.
type Program struct {
	Stage        Stage
	Major, Minor uint32
	Instructions []Instruction
}

// Instruction is one decoded instruction or declaration.
type Instruction struct {
	Op Opcode
	// Token is the opcode token, whose upper bits carry the opcode's
	// controls: saturate, the test of a conditional, an interpolation mode, a
	// resource dimension, global flags…
	Token    uint32
	Extended []uint32 // extended opcode tokens: sample offsets, resource dimension and return type
	Operands []Operand
	// Payload is what a declaration holds after its operands (a count, a
	// stride, a return type token, a system value name), or the body of a
	// customdata block (an immediate constant buffer).
	Payload []uint32
}

// OperandType is D3D10_SB_OPERAND_TYPE.
type OperandType uint32

// Some of the operand types; the others are named by operandPrefixes.
const (
	OperandTemp          OperandType = 0
	OperandInput         OperandType = 1
	OperandOutput        OperandType = 2
	OperandIndexableTemp OperandType = 3
	OperandImmediate32   OperandType = 4
	OperandImmediate64   OperandType = 5
	OperandSampler       OperandType = 6
	OperandResource      OperandType = 7
	OperandCBuffer       OperandType = 8
	OperandICB           OperandType = 9
	OperandLabel         OperandType = 10
	OperandNull          OperandType = 13
	OperandUAV           OperandType = 30
	OperandTGSM          OperandType = 31
)

// Selection is how a four-component operand picks its components.
type Selection uint8

// The selection modes.
const (
	SelectMask Selection = iota
	SelectSwizzle
	SelectOne
)

// Modifier is an operand's source modifier.
type Modifier uint8

// The source modifiers.
const (
	ModNone Modifier = iota
	ModNeg
	ModAbs
	ModAbsNeg
)

// Operand is one decoded operand.
type Operand struct {
	Type       OperandType
	Components int // 0, 1 or 4 (N-component operands are rejected)
	Selection  Selection
	Mask       uint8    // SelectMask: x=1 y=2 z=4 w=8
	Swizzle    [4]uint8 // SelectSwizzle: the source component of each; SelectOne: [0]
	Index      []Index
	Immediate  []uint32 // an immediate operand's values: 1 or 4 (or 2 or 8 for 64-bit)
	Modifier   Modifier
	Precision  uint32 // min precision (D3D11_SB_OPERAND_MIN_PRECISION)
	NonUniform bool
}

// Index is one dimension of an operand's index: an immediate, a relative
// operand, or their sum.
type Index struct {
	Imm      uint64
	Relative *Operand
}

// DecodeProgram decodes a SHDR or SHEX chunk.
func DecodeProgram(b []byte) (*Program, error) {
	if len(b) < 8 || len(b)%4 != 0 {
		return nil, fmt.Errorf("program chunk of %d bytes", len(b))
	}
	words := make([]uint32, len(b)/4)
	for i := range words {
		words[i] = le.Uint32(b[4*i:])
	}
	ver, n := words[0], words[1]
	if n < 2 || uint64(n) > uint64(len(words)) {
		return nil, fmt.Errorf("program claims %d tokens, chunk has %d", n, len(words))
	}
	p := &Program{Stage: Stage(ver >> 16), Major: ver >> 4 & 0xF, Minor: ver & 0xF}
	d := &decoder{w: words[:n], pos: 2}
	for d.pos < len(d.w) {
		in, err := d.instruction()
		if err != nil {
			return nil, fmt.Errorf("instruction at token %d: %w", d.pos, err)
		}
		p.Instructions = append(p.Instructions, in)
	}
	return p, nil
}

// decoder is a cursor over a program's tokens.
type decoder struct {
	w   []uint32
	pos int
	end int // the end of the instruction being decoded
}

var errTruncated = errors.New("runs past the program")

func (d *decoder) next() (uint32, error) {
	if d.pos >= d.end {
		return 0, errTruncated
	}
	v := d.w[d.pos]
	d.pos++
	return v, nil
}

func (d *decoder) instruction() (Instruction, error) {
	start := d.pos
	tok := d.w[start]
	in := Instruction{Op: Opcode(tok & 0x7FF), Token: tok}
	if int(in.Op) >= len(opInfos) {
		return in, fmt.Errorf("unknown opcode %d", in.Op)
	}
	length := int(tok >> 24 & 0x7F)
	if in.Op == OpCustomdata {
		if start+1 >= len(d.w) {
			return in, errTruncated
		}
		length = int(d.w[start+1]) // a customdata block counts its own tokens
		if length < 2 || start+length > len(d.w) {
			return in, fmt.Errorf("customdata of %d tokens %v", length, errTruncated)
		}
		in.Payload = d.w[start+2 : start+length]
		d.pos = start + length
		return in, nil
	}
	if length == 0 || start+length > len(d.w) {
		return in, fmt.Errorf("length %d %v", length, errTruncated)
	}
	d.pos, d.end = start+1, start+length
	if tok&0x80000000 != 0 {
		if err := d.extended(&in); err != nil {
			return in, err
		}
	}
	operands := opInfos[in.Op].operands
	for d.pos < d.end && (operands < 0 || len(in.Operands) < operands) {
		op, err := d.operand()
		if err != nil {
			return in, err
		}
		in.Operands = append(in.Operands, op)
	}
	in.Payload = d.w[d.pos:d.end]
	d.pos = d.end
	return in, nil
}

// extended reads the chain of extended opcode tokens.
func (d *decoder) extended(in *Instruction) error {
	for {
		t, err := d.next()
		if err != nil {
			return err
		}
		in.Extended = append(in.Extended, t)
		if t&0x80000000 == 0 {
			return nil
		}
	}
}

func (d *decoder) operand() (Operand, error) {
	t, err := d.next()
	if err != nil {
		return Operand{}, err
	}
	op := Operand{Type: OperandType(t >> 12 & 0xFF)}
	switch t & 3 {
	case 0:
		op.Components = 0
	case 1:
		op.Components = 1
	case 2:
		op.Components = 4
		op.Selection = Selection(t >> 2 & 3)
		switch op.Selection {
		case SelectMask:
			op.Mask = uint8(t >> 4 & 0xF)
		case SelectSwizzle:
			for i := range op.Swizzle {
				op.Swizzle[i] = uint8(t >> (4 + 2*i) & 3)
			}
		case SelectOne:
			op.Swizzle[0] = uint8(t >> 4 & 3)
		default:
			return op, fmt.Errorf("selection mode %d", op.Selection)
		}
	default:
		return op, errors.New("an N-component operand")
	}
	if t&0x80000000 != 0 {
		if err := d.operandExtension(&op); err != nil {
			return op, err
		}
	}
	if err := d.immediates(&op); err != nil {
		return op, err
	}
	dims := int(t >> 20 & 3)
	for i := range dims {
		idx, err := d.index(t >> (22 + 3*i) & 7)
		if err != nil {
			return op, err
		}
		op.Index = append(op.Index, idx)
	}
	return op, nil
}

// operandExtension reads the extended operand tokens: modifiers, minimum
// precision, non-uniform.
func (d *decoder) operandExtension(op *Operand) error {
	for {
		x, err := d.next()
		if err != nil {
			return err
		}
		if x&0x3F == 1 { // D3D10_SB_EXTENDED_OPERAND_MODIFIER
			op.Modifier = Modifier(x >> 6 & 0xFF)
			op.Precision = x >> 14 & 7
			op.NonUniform = x>>17&1 != 0
		}
		if x&0x80000000 == 0 {
			return nil
		}
	}
}

// immediates reads an immediate operand's values, which follow its token.
func (d *decoder) immediates(op *Operand) error {
	n := 0
	switch op.Type {
	case OperandImmediate32:
		n = 1
	case OperandImmediate64:
		n = 2
	default:
		return nil
	}
	if op.Components == 4 {
		n *= 4
	}
	for range n {
		v, err := d.next()
		if err != nil {
			return err
		}
		op.Immediate = append(op.Immediate, v)
	}
	return nil
}

func (d *decoder) index(rep uint32) (Index, error) {
	var idx Index
	switch rep {
	case 0, 3: // immediate32, immediate32 + relative
		v, err := d.next()
		if err != nil {
			return idx, err
		}
		idx.Imm = uint64(v)
	case 1, 4: // immediate64, immediate64 + relative
		hi, err := d.next()
		if err != nil {
			return idx, err
		}
		lo, err := d.next()
		if err != nil {
			return idx, err
		}
		idx.Imm = uint64(hi)<<32 | uint64(lo)
	case 2:
	default:
		return idx, fmt.Errorf("index representation %d", rep)
	}
	if rep >= 2 {
		rel, err := d.operand()
		if err != nil {
			return idx, err
		}
		idx.Relative = &rel
	}
	return idx, nil
}
