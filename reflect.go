package t7dxbc

import (
	"errors"
	"fmt"
)

// Reflection is what the RDEF chunk records about a shader: its constant
// buffers (and the element layout of its structured buffers), the resources
// bound to it, and which compiler built it.
type Reflection struct {
	Major, Minor uint8
	Program      uint16 // 0xFFFF pixel, 0xFFFE vertex, 0x4353 compute, …
	Flags        uint32
	Creator      string
	Buffers      []Buffer
	Bindings     []Binding
}

// BufferKind is D3D_CBUFFER_TYPE.
type BufferKind uint32

// The kinds of Buffer.
const (
	CBuffer BufferKind = iota
	TBuffer
	InterfacePointers
	ResourceBindInfo // the element layout of a structured buffer, named after it
)

// Buffer is a constant buffer, or the element layout of a structured buffer.
type Buffer struct {
	Name  string
	Kind  BufferKind
	Size  uint32
	Flags uint32
	Vars  []Variable
}

// VariableUsed is D3D_SVF_USED: the program reads the variable.
const VariableUsed = 2

// Variable is one member of a Buffer.
type Variable struct {
	Name   string
	Offset uint32
	Size   uint32
	Flags  uint32
	Type   *Type
}

// Type describes a variable's type: its class (scalar, vector, matrix, struct
// …), base type, shape, array length and, for a struct, its members.
type Type struct {
	Class    VarClass
	Base     VarType
	Rows     uint16
	Cols     uint16
	Elements uint16 // array length; 0 for a non-array
	Name     string // recorded from Shader Model 5 on ("float4", "Light")
	Members  []Member
}

// HLSLName is the type as HLSL spells it, from its shape: float4, uint,
// float4x3, or the struct's name. (The name the compiler records can be a
// legacy alias, dword for uint.)
func (t *Type) HLSLName() string {
	switch t.Class {
	case ClassStruct:
		return t.Name
	case ClassVector:
		return fmt.Sprintf("%s%d", t.Base, t.Cols)
	case ClassMatrixRows, ClassMatrixColumns:
		return fmt.Sprintf("%s%dx%d", t.Base, t.Rows, t.Cols)
	}
	return t.Base.String()
}

// Member is one field of a struct Type.
type Member struct {
	Name   string
	Offset uint32
	Type   *Type
}

// Binding is one resource the shader binds: a constant buffer, texture,
// sampler, structured or byte-address buffer, or UAV.
type Binding struct {
	Name       string
	Input      InputType
	Return     ReturnType
	Dim        Dimension
	NumSamples uint32 // the element stride for a structured buffer
	Point      uint32
	Count      uint32
	Flags      uint32
	Space, ID  uint32 // Shader Model 5.1
}

// rdef is a cursor over the RDEF chunk: every offset in it is from the
// chunk's start.
type rdef struct {
	b     []byte
	sm5   bool
	types map[uint32]*Type
	depth int
}

// maxTypeDepth bounds nested struct types, against a type whose member points
// back at itself.
const maxTypeDepth = 32

func (r *rdef) u32(off uint32) (uint32, error) {
	if uint64(off)+4 > uint64(len(r.b)) {
		return 0, fmt.Errorf("RDEF read at %d past its %d bytes", off, len(r.b))
	}
	return le.Uint32(r.b[off:]), nil
}

func (r *rdef) u16(off uint32) (uint16, error) {
	if uint64(off)+2 > uint64(len(r.b)) {
		return 0, fmt.Errorf("RDEF read at %d past its %d bytes", off, len(r.b))
	}
	return le.Uint16(r.b[off:]), nil
}

// str reads the NUL-terminated string at off.
func (r *rdef) str(off uint32) (string, error) {
	if uint64(off) >= uint64(len(r.b)) {
		return "", fmt.Errorf("RDEF string at %d past its %d bytes", off, len(r.b))
	}
	for i := off; i < uint32(len(r.b)); i++ {
		if r.b[i] == 0 {
			return string(r.b[off:i]), nil
		}
	}
	return "", fmt.Errorf("RDEF string at %d isn't terminated", off)
}

// table checks that n records of size bytes from at lie inside the chunk, in
// 64 bits: at + i*size can't wrap past zero once this holds.
func (r *rdef) table(at, n, size uint32, what string) error {
	if uint64(at)+uint64(n)*uint64(size) > uint64(len(r.b)) {
		return fmt.Errorf("%d %s from offset %d don't fit the %d-byte chunk", n, what, at, len(r.b))
	}
	return nil
}

// words reads n consecutive uint32 at off.
func (r *rdef) words(off uint32, n int) ([]uint32, error) {
	out := make([]uint32, n)
	for i := range out {
		v, err := r.u32(off + uint32(4*i))
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// ReadReflection reads an RDEF chunk.
func ReadReflection(b []byte) (*Reflection, error) {
	r := &rdef{b: b, types: map[uint32]*Type{}}
	h, err := r.words(0, 7)
	if err != nil {
		return nil, errors.New("RDEF header is truncated")
	}
	ref := &Reflection{Minor: b[16], Major: b[17], Program: le.Uint16(b[18:]), Flags: h[5]}
	r.sm5 = ref.Major >= 5
	if ref.Creator, err = r.str(h[6]); err != nil {
		return nil, err
	}
	bindSize := uint32(32)
	if r.sm5 {
		magic, err := r.u32(28)
		if err != nil || magic != 0x31314452 { // "RD11"
			return nil, errors.New("RDEF of shader model 5 without its RD11 header")
		}
		if sz, err := r.u32(40); err == nil && sz == 40 {
			bindSize = 40 // Shader Model 5.1 adds the register space and ID
		}
	}
	if ref.Bindings, err = r.bindings(h[3], h[2], bindSize); err != nil {
		return nil, err
	}
	if ref.Buffers, err = r.buffers(h[1], h[0]); err != nil {
		return nil, err
	}
	return ref, nil
}

func (r *rdef) bindings(at, n, size uint32) ([]Binding, error) {
	if err := r.table(at, n, size, "bindings"); err != nil {
		return nil, err
	}
	out := make([]Binding, n)
	for i := range out {
		w, err := r.words(at+uint32(i)*size, int(size/4))
		if err != nil {
			return nil, err
		}
		bd := Binding{Input: InputType(w[1]), Return: ReturnType(w[2]), Dim: Dimension(w[3]), NumSamples: w[4],
			Point: w[5], Count: w[6], Flags: w[7]}
		if size == 40 {
			bd.Space, bd.ID = w[8], w[9]
		}
		if bd.Name, err = r.str(w[0]); err != nil {
			return nil, err
		}
		out[i] = bd
	}
	return out, nil
}

func (r *rdef) buffers(at, n uint32) ([]Buffer, error) {
	if err := r.table(at, n, 24, "constant buffers"); err != nil {
		return nil, err
	}
	out := make([]Buffer, n)
	for i := range out {
		w, err := r.words(at+uint32(i)*24, 6)
		if err != nil {
			return nil, err
		}
		buf := Buffer{Size: w[3], Flags: w[4], Kind: BufferKind(w[5])}
		if buf.Name, err = r.str(w[0]); err != nil {
			return nil, err
		}
		if buf.Vars, err = r.variables(w[2], w[1]); err != nil {
			return nil, fmt.Errorf("cbuffer %s: %w", buf.Name, err)
		}
		out[i] = buf
	}
	return out, nil
}

func (r *rdef) variables(at, n uint32) ([]Variable, error) {
	size := uint32(24)
	if r.sm5 {
		size = 40 // adds the texture and sampler ranges of interface variables
	}
	if err := r.table(at, n, size, "variables"); err != nil {
		return nil, err
	}
	out := make([]Variable, n)
	for i := range out {
		w, err := r.words(at+uint32(i)*size, 5)
		if err != nil {
			return nil, err
		}
		v := Variable{Offset: w[1], Size: w[2], Flags: w[3]}
		if v.Name, err = r.str(w[0]); err != nil {
			return nil, err
		}
		if v.Type, err = r.typeAt(w[4]); err != nil {
			return nil, fmt.Errorf("variable %s: %w", v.Name, err)
		}
		out[i] = v
	}
	return out, nil
}

// typeAt reads the type description at off, once: types are shared.
func (r *rdef) typeAt(off uint32) (*Type, error) {
	if t, ok := r.types[off]; ok {
		return t, nil
	}
	if r.depth >= maxTypeDepth {
		return nil, errors.New("types nest too deep")
	}
	r.depth++
	defer func() { r.depth-- }()
	var f [6]uint16
	for i := range f {
		v, err := r.u16(off + uint32(2*i))
		if err != nil {
			return nil, err
		}
		f[i] = v
	}
	t := &Type{Class: VarClass(f[0]), Base: VarType(f[1]), Rows: f[2], Cols: f[3], Elements: f[4]}
	r.types[off] = t
	memberAt, err := r.u32(off + 12)
	if err != nil {
		return nil, err
	}
	if r.sm5 {
		nameAt, err := r.u32(off + 32)
		if err != nil {
			return nil, err
		}
		if nameAt != 0 {
			if t.Name, err = r.str(nameAt); err != nil {
				return nil, err
			}
		}
	}
	if t.Members, err = r.members(memberAt, uint32(f[5])); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *rdef) members(at, n uint32) ([]Member, error) {
	if err := r.table(at, n, 12, "struct members"); err != nil {
		return nil, err
	}
	out := make([]Member, n)
	for i := range out {
		w, err := r.words(at+uint32(i)*12, 3)
		if err != nil {
			return nil, err
		}
		m := Member{Offset: w[2]}
		if m.Name, err = r.str(w[0]); err != nil {
			return nil, err
		}
		if m.Type, err = r.typeAt(w[1]); err != nil {
			return nil, fmt.Errorf("member %s: %w", m.Name, err)
		}
		out[i] = m
	}
	return out, nil
}
