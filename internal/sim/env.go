package sim

import (
	"math"
	"strings"

	"github.com/McReaper/dxbc"
)

// NewWorld fills a shader's constant buffers from a seed: each variable with
// values of its type, floats in [-1, 2], integers in [0, 8), bools 0 or 1.
func NewWorld(ref *dxbc.Reflection, seed uint32) *World { return newWorld(ref, seed, false) }

// NewCountingWorld fills a shader's constant buffers as NewWorld does, but
// with integers in [0, 8) in the float variables too: a program that reads a
// loop count from one (asuint) ends.
func NewCountingWorld(ref *dxbc.Reflection, seed uint32) *World { return newWorld(ref, seed, true) }

func newWorld(ref *dxbc.Reflection, seed uint32, counting bool) *World {
	w := &World{CB: map[uint32][]uint32{}, Seed: seed, UAV: map[Cell]uint32{}, counters: map[uint32]uint32{},
		counting: counting}
	buffers := map[string]*dxbc.Buffer{}
	for i := range ref.Buffers {
		buffers[ref.Buffers[i].Name] = &ref.Buffers[i]
	}
	for _, b := range ref.Bindings {
		buf := buffers[b.Name]
		if b.Input != dxbc.InputCBuffer || buf == nil {
			continue
		}
		d := make([]uint32, (buf.Size+15)/16*4)
		for i := range d {
			d[i] = w.value(dxbc.TypeFloat, b.Point, uint32(i))
		}
		for _, v := range buf.Vars {
			w.fill(d, v.Offset, v.Type, b.Point)
		}
		w.CB[b.Point] = d
	}
	return w
}

// value is a constant of a base type.
func (w *World) value(t dxbc.VarType, slot, at uint32) uint32 {
	h := w.mix(0xCB, slot, at)
	switch {
	case t == dxbc.TypeBool:
		return h % 2
	case w.counting, t == dxbc.TypeInt, t == dxbc.TypeUint, t == dxbc.TypeUint8, t == dxbc.TypeMin16Int,
		t == dxbc.TypeMin12Int, t == dxbc.TypeMin16Uint:
		return h % 8
	}
	return math.Float32bits(float32(h%3001)/1000 - 1)
}

// fill writes the values of a variable of type t at byte offset off, in a
// constant buffer's layout: each array element and matrix line starts a
// register.
func (w *World) fill(d []uint32, off uint32, t *dxbc.Type, slot uint32) {
	if t.Elements > 0 {
		e := *t
		e.Elements = 0
		stride := (packedSize(&e) + 15) &^ 15
		for i := range uint32(t.Elements) {
			w.fill(d, off+i*stride, &e, slot)
		}
		return
	}
	put := func(at uint32) {
		if i := at / 4; i < uint32(len(d)) {
			d[i] = w.value(t.Base, slot, i)
		}
	}
	switch t.Class {
	case dxbc.ClassScalar:
		put(off)
	case dxbc.ClassVector:
		for c := range uint32(t.Cols) {
			put(off + 4*c)
		}
	case dxbc.ClassMatrixRows, dxbc.ClassMatrixColumns:
		lines, n := uint32(t.Rows), uint32(t.Cols)
		if t.Class == dxbc.ClassMatrixColumns {
			lines, n = n, lines
		}
		for l := range lines {
			for c := range n {
				put(off + 16*l + 4*c)
			}
		}
	case dxbc.ClassStruct:
		for _, m := range t.Members {
			w.fill(d, off+m.Offset, m.Type, slot)
		}
	}
}

// packedSize is a type's size in a constant buffer, its last line unpadded.
func packedSize(t *dxbc.Type) uint32 {
	var one uint32
	switch t.Class {
	case dxbc.ClassScalar:
		one = 4
	case dxbc.ClassVector:
		one = 4 * uint32(t.Cols)
	case dxbc.ClassMatrixRows, dxbc.ClassMatrixColumns:
		lines, n := uint32(t.Rows), uint32(t.Cols)
		if t.Class == dxbc.ClassMatrixColumns {
			lines, n = n, lines
		}
		one = 16*(lines-1) + 4*n
	case dxbc.ClassStruct:
		for _, m := range t.Members {
			one = max(one, m.Offset+packedSize(m.Type))
		}
	}
	if t.Elements > 0 {
		return (one+15)&^15*uint32(t.Elements-1) + one
	}
	return one
}

// Inputs makes an invocation's inputs from a seed: each element of the
// signature with values of its type and meaning. A position is a pixel's,
// texture coordinates are in [0, 1], other floats in [-1, 1], integers in
// [0, 8), the front face is either.
func Inputs(sig []dxbc.Element, seed uint32) map[uint32]Value {
	w := &World{Seed: seed}
	in := map[uint32]Value{}
	for _, el := range sig {
		if el.Register == dxbc.NoRegister {
			continue
		}
		v := in[el.Register]
		for c := range uint32(4) {
			if el.Mask&(1<<c) == 0 {
				continue
			}
			v[c] = w.input(el, c)
		}
		in[el.Register] = v
	}
	return in
}

func (w *World) input(el dxbc.Element, c uint32) uint32 {
	h := w.mix(0x1B, el.Register, c)
	name := strings.ToUpper(el.Name)
	switch {
	case name == "SV_ISFRONTFACE":
		return mask32(h%2 == 0)
	case el.Type != dxbc.ComponentFloat:
		return h % 8
	case name == "SV_POSITION" && c < 2:
		return math.Float32bits(float32(h%1900) + 0.5)
	case name == "SV_POSITION":
		return math.Float32bits(float32(h%1000)/1000 + float32(c-2)*0.5)
	case strings.HasPrefix(name, "TEXCOORD"):
		return math.Float32bits(float32(h%1001) / 1000)
	}
	return math.Float32bits(float32(h%2001)/1000 - 1)
}
