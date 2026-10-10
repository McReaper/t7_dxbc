// Command t7dxbc disassembles or decompiles compiled Direct3D shaders.
//
//	t7dxbc disasm <file>      the instructions, as Microsoft's disassembler writes them
//	t7dxbc decompile <file>   HLSL that recompiles to the same interface
package main

import (
	"fmt"
	"os"

	"github.com/McReaper/t7dxbc"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: t7dxbc disasm|decompile <file>")
		os.Exit(2)
	}
	code, err := os.ReadFile(os.Args[2])
	if err != nil {
		fail(err)
	}
	switch os.Args[1] {
	case "disasm":
		c, err := t7dxbc.Parse(code)
		if err != nil {
			fail(err)
		}
		p, err := t7dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
		if err != nil {
			fail(err)
		}
		for _, l := range p.Disassemble() {
			fmt.Println(l)
		}
	case "decompile":
		src, err := t7dxbc.Decompile(code)
		fmt.Print(src)
		if err != nil {
			fail(err)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: t7dxbc disasm|decompile <file>")
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "t7dxbc:", err)
	os.Exit(1)
}
