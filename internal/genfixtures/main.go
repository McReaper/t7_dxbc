//go:build windows

// Command genfixtures compiles testdata/*.hlsl with the Windows HLSL compiler
// into testdata/<name>.dxbc, and writes Microsoft's disassembly of each beside
// it (<name>.asm), the golden output the decoder is checked against on any OS.
// The first line of each source names its entry point and target:
// "// dxbc: main ps_5_0". Run from the module root: go run ./internal/genfixtures
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/McReaper/dxbc/internal/d3dc"
)

func main() {
	srcs, _ := filepath.Glob(filepath.Join("testdata", "*.hlsl"))
	for _, src := range srcs {
		if err := gen(src); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func gen(src string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	first, _, _ := strings.Cut(string(b), "\n")
	fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(first), "// dxbc:"))
	if len(fields) != 2 {
		return fmt.Errorf("%s: first line must be // dxbc: <entry> <target>", src)
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		return err
	}
	code, err := d3dc.Compile(b, abs, fields[0], fields[1])
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	asm, err := d3dc.Disassemble(code)
	if err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	base := strings.TrimSuffix(src, ".hlsl")
	if err := os.WriteFile(base+".dxbc", code, 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", base+".dxbc", len(code), "bytes")
	return os.WriteFile(base+".asm", []byte(asm), 0o644)
}
