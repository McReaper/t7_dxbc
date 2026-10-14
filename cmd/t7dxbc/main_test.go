package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/McReaper/t7_dxbc/dxbc"
	"github.com/McReaper/t7_dxbc/internal/fixture"
)

// TestMain runs the command itself when a test starts this binary again
// with T7DXBC_MAIN set: main is tested through its real arguments.
func TestMain(m *testing.M) {
	if os.Getenv("T7DXBC_MAIN") != "" {
		main()
	}
	os.Exit(m.Run())
}

// main hands the command line, without the program's name, to run.
func TestMainTakesItsArguments(t *testing.T) {
	cmd := exec.Command(os.Args[0], "decompile", filepath.Join(fixture.Dir(), "ps_basic.dxbc"))
	cmd.Env = append(os.Environ(), "T7DXBC_MAIN=1")
	out, err := cmd.Output()
	if err != nil || !strings.HasPrefix(string(out), "// Decompiled by") {
		t.Errorf("%v: %.60q", err, out)
	}
}

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
	if code != 1 || !strings.Contains(out, "void ps_main(") ||
		!strings.HasPrefix(errs, "t7dxbc: not decompiled: cb0 without reflection") {
		t.Errorf("exit %d, stderr %q, stdout\n%s", code, errs, out)
	}
}

func TestVersion(t *testing.T) {
	if code, out, errs := runWith("version"); code != 0 || errs != "" || out != "t7dxbc "+release(moduleVersion())+"\n" {
		t.Errorf("exit %d, stderr %q, stdout %q", code, errs, out)
	}
	for module, want := range map[string]string{
		"v0.2.0":                             "0.2.0",
		"v0.0.0-20261010154416-eda05ce43426": "0.0.0-20261010154416-eda05ce43426",
		"(devel)":                            "devel",
		"":                                   "devel",
	} {
		if got := release(module); got != want {
			t.Errorf("module %q: %q, want %q", module, got, want)
		}
	}
	version = "1.2.3"
	defer func() { version = "" }()
	if got := release("v0.2.0"); got != "1.2.3" {
		t.Errorf("the release build's version: %q", got)
	}
}

// The release build's -X flag, as .github/goreleaser.yaml writes it, sets the
// version the command prints.
func TestReleaseBuildSetsVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the command")
	}
	config, err := os.ReadFile(filepath.Join("..", "..", ".github", "goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`-X (\S+)=\{\{ \.Version \}\}`).FindSubmatch(config)
	if m == nil {
		t.Fatal("no -X <var>={{ .Version }} in the release config")
	}
	bin := filepath.Join(t.TempDir(), "t7dxbc")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-ldflags", "-X "+string(m[1])+"=1.2.3", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil || string(out) != "t7dxbc 1.2.3\n" {
		t.Errorf("%v: %q", err, out)
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
		{[]string{"version", notShader}, 2, "usage:"},
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
