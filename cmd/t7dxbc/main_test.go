package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/McReaper/t7dxbc/dxbc"
	"github.com/McReaper/t7dxbc/internal/fixture"
)

func runWith(args ...string) (code int, stdout, stderr string) {
	var out, errs bytes.Buffer
	code = run(args, &out, &errs)
	return code, out.String(), errs.String()
}

func TestCommands(t *testing.T) {
	shader := filepath.Join(fixture.Dir(), "ps_basic.dxbc")
	golden, err := os.ReadFile(filepath.Join(fixture.Dir(), "ps_basic.decompiled.hlsl"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs := runWith("decompile", shader)
	if code != 0 || errs != "" || out != strings.ReplaceAll(string(golden), "\r\n", "\n") {
		t.Errorf("decompile: exit %d, stderr %q, stdout the golden: %v", code, errs, out == string(golden))
	}
	c, err := dxbc.Parse(fixture.Shaders(t)["ps_basic.dxbc"][0])
	if err != nil {
		t.Fatal(err)
	}
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs = runWith("disasm", shader)
	if want := strings.Join(p.Disassemble(), "\n") + "\n"; code != 0 || errs != "" || out != want {
		t.Errorf("disasm: exit %d, stderr %q, stdout\n%s\nwant\n%s", code, errs, out, want)
	}
}

// A shader the decompiler can't finish prints what it wrote, and the error.
func TestPartialDecompilation(t *testing.T) {
	b := bytes.Replace(fixture.Shaders(t)["ps_basic.dxbc"][0], []byte("RDEF"), []byte("XDEF"), 1)
	stripped := filepath.Join(t.TempDir(), "stripped.dxbc")
	if err := os.WriteFile(stripped, b, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errs := runWith("decompile", stripped)
	if code != 1 || !strings.Contains(out, "void main(") ||
		!strings.HasPrefix(errs, "t7dxbc: not decompiled: cb0 without reflection") {
		t.Errorf("exit %d, stderr %q, stdout\n%s", code, errs, out)
	}
}

func TestErrors(t *testing.T) {
	notShader := filepath.Join(t.TempDir(), "x.dxbc")
	if err := os.WriteFile(notShader, []byte("not a shader"), 0o644); err != nil {
		t.Fatal(err)
	}
	noProgram := filepath.Join(t.TempDir(), "y.dxbc")
	if err := os.WriteFile(noProgram, fixture.Container(fixture.Chunk{FourCC: "RDEF"}), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args   []string
		code   int
		stderr string
	}{
		{nil, 2, "usage:"},
		{[]string{"disasm"}, 2, "usage:"},
		{[]string{"run", notShader}, 2, "usage:"},
		{[]string{"disasm", notShader, "extra"}, 2, "usage:"},
		{[]string{"disasm", filepath.Join(t.TempDir(), "missing")}, 1, "t7dxbc: open "},
		{[]string{"disasm", notShader}, 1, "t7dxbc: "},
		{[]string{"disasm", noProgram}, 1, "t7dxbc: "},
		{[]string{"decompile", notShader}, 1, "t7dxbc: "},
		{[]string{"decompile", noProgram}, 1, "t7dxbc: no SHEX"},
	}
	for _, c := range cases {
		code, out, errs := runWith(c.args...)
		if code != c.code || !strings.HasPrefix(errs, c.stderr) || code == 2 && out != "" {
			t.Errorf("%q: exit %d, stderr %q, stdout %q; want exit %d, stderr %q…", c.args, code, errs, out, c.code, c.stderr)
		}
	}
}
