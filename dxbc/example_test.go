package dxbc_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/McReaper/t7_dxbc/dxbc"
)

// The disassembly of a vertex shader: its declarations, then its
// instructions, as Microsoft's disassembler writes them.
func ExampleProgram_Disassemble() {
	code, err := os.ReadFile(filepath.Join("..", "testdata", "vs_taps.dxbc"))
	if err != nil {
		log.Fatal(err)
	}
	c, err := dxbc.Parse(code)
	if err != nil {
		log.Fatal(err)
	}
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		log.Fatal(err)
	}
	for _, line := range p.Disassemble()[:6] {
		fmt.Println(line)
	}
	// Output:
	// dcl_globalFlags refactoringAllowed
	// dcl_constantbuffer CB0[1], immediateIndexed
	// dcl_input v0.xyzw
	// dcl_input v1.xy
	// dcl_output_siv o0.xyzw, position
	// dcl_output o1.xy
}
