package dxbc

import (
	"fmt"
	"strings"
)

// Name is the opcode's mnemonic: add, dcl_temps.
func (o Opcode) Name() string {
	if int(o) < len(opInfos) {
		return opInfos[o].name
	}
	return fmt.Sprintf("opcode%d", uint32(o))
}

// IsDeclaration reports whether the opcode declares something (a register, a
// resource, the immediate constant buffer) rather than computes.
func (o Opcode) IsDeclaration() bool {
	if int(o) >= len(opInfos) {
		return false
	}
	return o == OpCustomdata || opInfos[o].operands >= 0 || strings.HasPrefix(opInfos[o].name, "dcl_")
}
