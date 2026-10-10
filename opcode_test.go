package dxbc

import "testing"

func TestOpcodeNames(t *testing.T) {
	cases := []struct {
		op   Opcode
		name string
		decl bool
	}{
		{OpAdd, "add", false}, {OpDclTemps, "dcl_temps", true}, {OpCustomdata, "customdata", true},
		{OpDclGlobalFlags, "dcl_global_flags", true}, {OpLdStructured, "ld_structured", false},
		{Opcode(9999), "opcode9999", false},
	}
	for _, c := range cases {
		if got := c.op.Name(); got != c.name {
			t.Errorf("Name(%d) = %q, want %q", c.op, got, c.name)
		}
		if got := c.op.IsDeclaration(); got != c.decl {
			t.Errorf("IsDeclaration(%s) = %v, want %v", c.name, got, c.decl)
		}
	}
}
