// Command t7dxbc disassembles or decompiles compiled Direct3D shaders.
//
//	t7dxbc disasm <file>      the instructions, as Microsoft's disassembler writes them
//	t7dxbc decompile <file>   HLSL that recompiles to the same interface
//
// A shader the decompiler can't finish still prints what it wrote, with the
// error that names what is missing.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/McReaper/t7dxbc/dxbc"
	"github.com/McReaper/t7dxbc/hlsl"
)

const usage = "usage: t7dxbc disasm|decompile <file>"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run carries out a command line and returns the exit code: 0, 1 for a
// shader that couldn't be read or decompiled, 2 for a wrong command line.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "disasm" && args[0] != "decompile" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	code, err := os.ReadFile(args[1])
	if err == nil {
		if args[0] == "disasm" {
			err = disassemble(code, stdout)
		} else {
			var src string
			src, err = hlsl.Decompile(code)
			fmt.Fprint(stdout, src)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, "t7dxbc:", err)
		return 1
	}
	return 0
}

func disassemble(code []byte, w io.Writer) error {
	c, err := dxbc.Parse(code)
	if err != nil {
		return err
	}
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return err
	}
	for _, l := range p.Disassemble() {
		fmt.Fprintln(w, l)
	}
	return nil
}
