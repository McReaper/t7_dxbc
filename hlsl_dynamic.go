package dxbc

import (
	"fmt"
	"strings"
)

// A structured buffer's dword at a byte offset computed at run time can't be
// named statically: it could be any member. The decompiled shader carries a
// helper per buffer and access that tests the offset against the element's
// layout, member by member, an array taking an index by division, so the
// access stays exact whatever the offset.

// dynamicAccess is the kind of access a helper makes.
type dynamicAccess struct {
	name   string // load, store, or the atomic intrinsic
	params []string
}

var (
	dynamicLoad  = dynamicAccess{name: "load"}
	dynamicStore = dynamicAccess{name: "store", params: []string{"dxbc_value"}}
)

// dynamicHelper returns the helper that makes an access to the elements of a
// structured view, writing it the first time.
func (e *emitter) dynamicHelper(acc dynamicAccess, res string, t *Type) string {
	name := fmt.Sprintf("dxbc_%s_%s", acc.name, identifier(res))
	if _, ok := e.generated[name]; ok {
		return name
	}
	var sb strings.Builder
	params := []string{"uint dxbc_i", "uint dxbc_o0"}
	for _, p := range acc.params {
		params = append(params, "uint "+p)
	}
	ret := "uint"
	if acc.name == "store" {
		ret = "void"
	}
	fmt.Fprintf(&sb, "%s %s(%s)\n{\n", ret, name, strings.Join(params, ", "))
	if ret == "uint" {
		sb.WriteString("  uint dxbc_old = 0;\n")
	}
	g := dwordGen{sb: &sb, acc: acc}
	g.value(res+"[dxbc_i]", t, "dxbc_o0", 1)
	if ret == "uint" {
		sb.WriteString("  return dxbc_old;\n")
	}
	sb.WriteString("}\n")
	e.generated[name] = sb.String()
	e.generatedOrder = append(e.generatedOrder, name)
	return name
}

// dwordGen writes the tests that find a dword in a value's layout.
type dwordGen struct {
	sb  *strings.Builder
	acc dynamicAccess
}

// value writes the tests for a value of type t named path, whose byte offset
// is in the variable o.
func (g dwordGen) value(path string, t *Type, o string, depth int) {
	ind := strings.Repeat("  ", depth)
	next := fmt.Sprintf("dxbc_o%d", depth)
	if t.Elements > 0 {
		elem := *t
		elem.Elements = 0
		stride := typeSize(&elem, false)
		if stride == 0 {
			return
		}
		k := fmt.Sprintf("dxbc_k%d", depth)
		fmt.Fprintf(g.sb, "%sif (%s < %du) {\n", ind, o, stride*uint32(t.Elements))
		fmt.Fprintf(g.sb, "%s  uint %s = %s / %du;\n%s  uint %s = %s %% %du;\n", ind, k, o, stride, ind, next, o, stride)
		g.value(fmt.Sprintf("%s[%s]", path, k), &elem, next, depth+1)
		fmt.Fprintf(g.sb, "%s}\n", ind)
		return
	}
	if t.Class == ClassStruct {
		for _, m := range t.Members {
			member := path + "." + identifier(m.Name)
			if m.Type.Class == ClassScalar && m.Type.Elements == 0 {
				if s, ok := scalarIn(member, m.Type, 0, false); ok {
					fmt.Fprintf(g.sb, "%sif (%s == %du) { %s }\n", ind, o, m.Offset, g.act(s))
				}
				continue
			}
			size := typeSize(m.Type, false)
			fmt.Fprintf(g.sb, "%sif (%s >= %du && %s < %du) {\n", ind, o, m.Offset, o, m.Offset+size)
			fmt.Fprintf(g.sb, "%s  uint %s = %s - %du;\n", ind, next, o, m.Offset)
			g.value(member, m.Type, next, depth+1)
			fmt.Fprintf(g.sb, "%s}\n", ind)
		}
		return
	}
	for off := uint32(0); off < typeSize(t, false); off += 4 {
		if s, ok := scalarIn(path, t, off, false); ok {
			fmt.Fprintf(g.sb, "%sif (%s == %du) { %s }\n", ind, o, off, g.act(s))
		}
	}
}

// act is what the helper does to the dword it found.
func (g dwordGen) act(s scalar) string {
	t := varValType(s.typ)
	isBool := s.typ == TypeBool
	switch g.acc.name {
	case "load":
		if isBool {
			return fmt.Sprintf("return %s ? 1u : 0u;", s.expr)
		}
		return fmt.Sprintf("return %s;", convert(s.expr, t, tUint))
	case "store":
		if isBool {
			return fmt.Sprintf("%s = dxbc_value != 0; return;", s.expr)
		}
		return fmt.Sprintf("%s = %s; return;", s.expr, convert("dxbc_value", tUint, t))
	}
	if t == tFloat || isBool {
		return "return 0;" // atomics change integers only
	}
	args := make([]string, len(g.acc.params))
	for i, p := range g.acc.params {
		args[i] = convert(p, tUint, t)
	}
	if g.acc.name == "InterlockedCompareStore" {
		return fmt.Sprintf("%s(%s, %s); return 0;", g.acc.name, s.expr, strings.Join(args, ", "))
	}
	return fmt.Sprintf("%s(%s, %s, dxbc_old); return dxbc_old;", g.acc.name, s.expr, strings.Join(args, ", "))
}

// dynamicDword is the dword c dwords past a computed byte offset of element
// idx of a structured view: read through the load helper or, for a store,
// written through the store one.
func (e *emitter) dynamicDword(op *Operand, idx string, offOp *Operand, c uint32, store bool) (scalar, bool) {
	res, b := e.resourceName(op)
	if b == nil {
		return scalar{}, false
	}
	t := e.structs[b.Name]
	if t == nil {
		e.unknown["computed offset into "+res+", whose element is unknown"] = true
		return scalar{}, false
	}
	off := e.read(offOp, picks(offOp, 1), tUint)
	if c > 0 {
		off = fmt.Sprintf("%s + %du", paren(off), 4*c)
	}
	args := idx + ", " + off
	if store {
		call := fmt.Sprintf("%s(%s, %%s)", e.dynamicHelper(dynamicStore, res, t), args)
		return scalar{typ: TypeUint, store: call}, true
	}
	load := fmt.Sprintf("%s(%s)", e.dynamicHelper(dynamicLoad, res, t), args)
	return scalar{expr: load, base: load, comp: -1, typ: TypeUint}, true
}
