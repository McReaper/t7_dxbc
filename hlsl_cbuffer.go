package dxbc

import (
	"fmt"
	"sort"
	"strings"
)

// cbufferLayout is a constant buffer's variables, and finds which one, and
// which component of it, a register component holds.
type cbufferLayout struct {
	buf  *Buffer
	vars []Variable // by offset
}

func newCBufferLayout(buf *Buffer) *cbufferLayout {
	vars := append([]Variable(nil), buf.Vars...)
	sort.SliceStable(vars, func(i, j int) bool { return vars[i].Offset < vars[j].Offset })
	return &cbufferLayout{buf: buf, vars: vars}
}

// scalar is one 32-bit component of a variable: the HLSL expression that
// reads it and the base type it has.
type scalar struct {
	expr string // e.g. tint.y, weights[1].z, key.color.x, uvTransform._m10
	base string // the expression of its innermost vector, for joining: tint, weights[1]
	comp int    // its component within base, or -1 if base isn't a vector
	typ  VarType
	// store, when set, is the call that writes the scalar, its value a %s:
	// a dword no expression names
	store string
}

// at finds the scalar at a byte offset of the buffer.
func (l *cbufferLayout) at(off uint32) (scalar, bool) {
	i := sort.Search(len(l.vars), func(i int) bool { return l.vars[i].Offset > off }) - 1
	if i < 0 {
		return scalar{}, false
	}
	v := l.vars[i]
	if off >= v.Offset+v.Size {
		return scalar{}, false
	}
	return scalarIn(identifier(v.Name), v.Type, off-v.Offset, true)
}

// scalarIn finds the scalar at off bytes into a value of type t, named name.
// In a constant buffer (packed) each array element and each matrix row or
// column starts a 16-byte register.
func scalarIn(name string, t *Type, off uint32, packed bool) (scalar, bool) {
	if t.Elements > 0 {
		elem := *t
		elem.Elements = 0
		stride := typeSize(&elem, packed)
		if packed {
			stride = (stride + 15) &^ 15
		}
		if stride == 0 {
			return scalar{}, false
		}
		i := off / stride
		if i >= uint32(t.Elements) {
			return scalar{}, false
		}
		return scalarIn(fmt.Sprintf("%s[%d]", name, i), &elem, off-i*stride, packed)
	}
	switch t.Class {
	case ClassScalar:
		return scalar{expr: name, base: name, comp: -1, typ: t.Base}, off == 0
	case ClassVector:
		c := int(off / 4)
		if c >= int(t.Cols) || off%4 != 0 {
			return scalar{}, false
		}
		return scalar{expr: name + "." + components[c:c+1], base: name, comp: c, typ: t.Base}, true
	case ClassMatrixRows, ClassMatrixColumns:
		return matrixScalar(name, t, off, packed)
	case ClassStruct:
		for i := len(t.Members) - 1; i >= 0; i-- {
			m := t.Members[i]
			if off >= m.Offset {
				return scalarIn(name+"."+identifier(m.Name), m.Type, off-m.Offset, packed)
			}
		}
	}
	return scalar{}, false
}

// matrixScalar finds a matrix element. A row_major matrix holds a row per
// register (or per rows×4 bytes unpacked), a column_major one a column.
func matrixScalar(name string, t *Type, off uint32, packed bool) (scalar, bool) {
	lineLen := uint32(t.Cols) // components per row
	if t.Class == ClassMatrixColumns {
		lineLen = uint32(t.Rows)
	}
	stride := lineLen * 4
	if packed {
		stride = 16
	}
	line, c := off/stride, (off%stride)/4
	if c >= lineLen || off%4 != 0 {
		return scalar{}, false
	}
	row, col := line, c
	if t.Class == ClassMatrixColumns {
		row, col = c, line
	}
	if row >= uint32(t.Rows) || col >= uint32(t.Cols) {
		return scalar{}, false
	}
	return scalar{expr: fmt.Sprintf("%s._m%d%d", name, row, col), comp: -1, typ: t.Base}, true
}

// typeSize is the bytes a value of type t takes: packed, the layout of a
// constant buffer (an array element or matrix line padded to 16 bytes, but
// the last); unpacked, a structured buffer's tight layout.
func typeSize(t *Type, packed bool) uint32 {
	var one uint32
	switch t.Class {
	case ClassScalar:
		one = 4
	case ClassVector:
		one = 4 * uint32(t.Cols)
	case ClassMatrixRows, ClassMatrixColumns:
		lines, n := uint32(t.Rows), uint32(t.Cols)
		if t.Class == ClassMatrixColumns {
			lines, n = n, lines
		}
		if packed {
			one = 16*(lines-1) + 4*n
		} else {
			one = 4 * lines * n
		}
	case ClassStruct:
		for _, m := range t.Members {
			if end := m.Offset + typeSize(m.Type, packed); end > one {
				one = end
			}
		}
	}
	if t.Elements == 0 {
		return one
	}
	stride := one
	if packed {
		stride = (one + 15) &^ 15
	}
	return stride*uint32(t.Elements-1) + one
}

// declType is a variable's type as a declaration spells it: row_major for a
// row-major matrix, the struct's name for a struct.
func declType(t *Type) string {
	name := t.HLSLName()
	if t.Class == ClassStruct {
		name = identifier(t.Name)
	}
	if t.Class == ClassMatrixRows {
		return "row_major " + name
	}
	return name
}

func arraySuffix(t *Type) string {
	if t.Elements > 0 {
		return fmt.Sprintf("[%d]", t.Elements)
	}
	return ""
}

// packOffset spells a variable's offset as a packoffset: c3, c3.z.
func packOffset(off uint32) string {
	s := fmt.Sprintf("c%d", off/16)
	if c := off % 16 / 4; c != 0 {
		s += "." + components[c:c+1]
	}
	return s
}

// identifier makes a reflected name a valid HLSL identifier: $Element and
// $Globals start with a dollar.
func identifier(name string) string {
	var sb strings.Builder
	for i, r := range name {
		ok := r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || i > 0 && r >= '0' && r <= '9'
		if ok {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('_')
		}
	}
	if sb.Len() == 0 {
		return "_"
	}
	return sb.String()
}
