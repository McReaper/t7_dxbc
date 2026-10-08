//go:build windows

package dxbc

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/McReaper/dxbc/internal/d3dc"
)

// stageTargets are the compile targets' prefixes by stage.
var stageTargets = map[Stage]string{StagePixel: "ps", StageVertex: "vs", StageCompute: "cs"}

// recompile decompiles a shader and compiles the HLSL back with Microsoft's
// compiler, for the same target.
func recompile(code []byte) (src string, out []byte, err error) {
	src, err = Decompile(code)
	if err != nil {
		return src, nil, fmt.Errorf("decompile: %w", err)
	}
	c, _ := Parse(code)
	p, _ := DecodeProgram(c.Chunk("SHEX", "SHDR"))
	target := fmt.Sprintf("%s_%d_%d", stageTargets[p.Stage], p.Major, p.Minor)
	out, err = d3dc.Compile([]byte(src), "decompiled.hlsl", "main", target)
	if err != nil {
		return src, nil, fmt.Errorf("compile: %w", err)
	}
	return src, out, nil
}

// shaderInterface is what a shader exposes to the engine that binds it: its
// constant buffers' layout and which variables it reads, its resources, and
// its signatures.
type shaderInterface struct {
	Buffers  []string
	Bindings []string
	Inputs   []string
	Outputs  []string
}

func interfaceOf(code []byte) (shaderInterface, error) {
	var s shaderInterface
	c, err := Parse(code)
	if err != nil {
		return s, err
	}
	ref := &Reflection{}
	if rdef := c.Chunk("RDEF"); rdef != nil {
		if ref, err = ReadReflection(rdef); err != nil {
			return s, err
		}
	}
	for _, b := range ref.Buffers {
		s.Buffers = append(s.Buffers, fmt.Sprintf("%s kind=%d size=%d", b.Name, b.Kind, b.Size))
		for _, v := range b.Vars {
			s.Buffers = append(s.Buffers, fmt.Sprintf("  %s %s @%d+%d used=%v", v.Type.HLSLName(), v.Name, v.Offset,
				v.Size, v.Flags&VariableUsed != 0))
		}
	}
	for _, b := range ref.Bindings {
		s.Bindings = append(s.Bindings, fmt.Sprintf("%s %s %s %s %s count=%d", b.Name, b.Kind(), b.Format(), b.DimName(),
			b.Register(), b.Count))
	}
	in, out, err := c.Signatures()
	if err != nil {
		return s, err
	}
	sig := func(els []Element) []string {
		var rows []string
		for _, el := range els {
			rows = append(rows, fmt.Sprintf("%s%d %v %v r%d %s", el.Name, el.Index, el.SystemValue, el.Type, el.Register,
				MaskString(el.Mask)))
		}
		return rows
	}
	s.Inputs, s.Outputs = sig(in), sig(out)
	return s, nil
}

// interfaceDiffs compares two shaders' interfaces.
func interfaceDiffs(want, got []byte) []string {
	a, err := interfaceOf(want)
	if err != nil {
		return []string{"original: " + err.Error()}
	}
	b, err := interfaceOf(got)
	if err != nil {
		return []string{"recompiled: " + err.Error()}
	}
	var diffs []string
	for _, f := range []struct {
		name   string
		wa, ga []string
	}{{"buffers", a.Buffers, b.Buffers}, {"bindings", a.Bindings, b.Bindings}, {"inputs", a.Inputs, b.Inputs},
		{"outputs", a.Outputs, b.Outputs}} {
		if !reflect.DeepEqual(f.wa, f.ga) {
			diffs = append(diffs, fmt.Sprintf("%s: %s\n  want %s\n  got  %s", f.name, firstDifference(f.wa, f.ga),
				strings.Join(f.wa, " | "), strings.Join(f.ga, " | ")))
		}
	}
	return diffs
}

// firstDifference names how two lists first differ: a variable whose use
// changed, a line missing or out of order.
func firstDifference(want, got []string) string {
	for i := range want {
		if i >= len(got) {
			return "missing " + strings.Fields(want[i])[0]
		}
		if want[i] == got[i] {
			continue
		}
		w, g := strings.Fields(want[i]), strings.Fields(got[i])
		if len(w) == len(g) && len(w) > 1 && strings.HasPrefix(w[len(w)-1], "used=") &&
			strings.Join(w[:len(w)-1], " ") == strings.Join(g[:len(g)-1], " ") {
			return "use changed (" + w[len(w)-1] + ")"
		}
		return "differs"
	}
	return "extra"
}

// statementCount is how many instructions a shader runs, declarations aside.
func statementCount(code []byte) int {
	c, err := Parse(code)
	if err != nil {
		return 0
	}
	p, err := DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return 0
	}
	n := 0
	for _, in := range p.Instructions {
		if opInfos[in.Op].operands < 0 && !strings.HasPrefix(opInfos[in.Op].name, "dcl_") && in.Op != OpCustomdata {
			n++
		}
	}
	return n
}

func TestDecompileRecompiles(t *testing.T) {
	if !d3dc.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	for name, f := range fixtures(t) {
		t.Run(name, func(t *testing.T) {
			src, out, err := recompile(f[0])
			if err != nil {
				t.Fatalf("%v\n%s", err, src)
			}
			if d := interfaceDiffs(f[0], out); len(d) > 0 {
				t.Errorf("interface differs:\n%s\n%s", strings.Join(d, "\n"), src)
			}
			t.Logf("instructions: %d original, %d recompiled", statementCount(f[0]), statementCount(out))
		})
	}
}

// TestCorpusDecompile is the opt-in oracle over a real shader cache: how many
// shaders decompile, recompile, and keep their interface.
func TestCorpusDecompile(t *testing.T) {
	paths := corpus(t)
	type result struct {
		path, kind string
		ratio      float64
		took       time.Duration
	}
	results := make([]result, len(paths))
	var wg sync.WaitGroup
	next := make(chan int)
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				results[i] = recompileResult(paths[i])
			}
		}()
	}
	for i := range paths {
		next <- i
	}
	close(next)
	wg.Wait()
	var total, decompiled, compiled, same int
	var ratio float64
	kinds := map[string]int{}
	examples := map[string]string{}
	for _, r := range results {
		if r.kind == "skip" {
			continue
		}
		total++
		if !strings.HasPrefix(r.kind, "decompile") {
			decompiled++
		}
		if r.kind == "" || strings.HasPrefix(r.kind, "interface") {
			compiled++
		}
		if r.kind == "" {
			same++
			ratio += r.ratio
			continue
		}
		kinds[r.kind]++
		if ex, ok := examples[r.kind]; !ok || fileSize(r.path) < fileSize(filepath.Join(filepath.Dir(r.path), ex)) {
			examples[r.kind] = filepath.Base(r.path) // the smallest, to read
		}
	}
	t.Logf("%d shaders: %d decompiled, %d recompiled, %d with the same interface", total, decompiled, compiled, same)
	if same > 0 {
		t.Logf("recompiled instruction count: %.2f× the original on average", ratio/float64(same))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].took > results[j].took })
	for _, r := range results[:min(3, len(results))] {
		t.Logf("slowest: %s %v", filepath.Base(r.path), r.took.Round(time.Millisecond))
	}
	keys := make([]string, 0, len(kinds))
	for k := range kinds {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return kinds[keys[i]] > kinds[keys[j]] })
	for i, k := range keys {
		if i == 40 {
			break
		}
		t.Logf("  %5d  %s  (e.g. %s)", kinds[k], k, examples[k])
	}
}

// recompileResult decompiles and recompiles one corpus file: kind is "" when
// the interface is kept, "skip" when the file isn't a shader, else what
// failed.
func recompileResult(path string) (r struct {
	path, kind string
	ratio      float64
	took       time.Duration
}) {
	r.path = path
	start := time.Now()
	defer func() { r.took = time.Since(start) }()
	code, err := os.ReadFile(path)
	if err != nil || len(code) < 4 || string(code[:4]) != "DXBC" {
		r.kind = "skip"
		return r
	}
	_, out, err := recompile(code)
	switch {
	case err != nil && strings.HasPrefix(err.Error(), "decompile"):
		r.kind = firstLine(err.Error())
		return r
	case err != nil:
		r.kind = "compile: " + compileErrorKind(err.Error())
		return r
	}
	if d := interfaceDiffs(code, out); len(d) > 0 {
		r.kind = "interface: " + firstLine(d[0])
		return r
	}
	if n := statementCount(code); n > 0 {
		r.ratio = float64(statementCount(out)) / float64(n)
	}
	return r
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	if len(s) > 160 {
		s = s[:160]
	}
	return s
}

// compileErrorKind is a compiler error without its position: "error X3004:
// undeclared identifier 'x'".
func compileErrorKind(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if i := strings.Index(l, "error X"); i >= 0 {
			msg := l[i:]
			if j := strings.Index(msg, "'"); j >= 0 {
				msg = msg[:j]
			}
			return msg
		}
	}
	return firstLine(s)
}

// TestDecompileOne shows one shader's decompilation, recompilation errors and
// interface differences: DXBC_ONE=<file>.
func TestDecompileOne(t *testing.T) {
	p := os.Getenv("DXBC_ONE")
	if p == "" {
		t.Skip("set DXBC_ONE to a compiled shader")
	}
	code, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	src, out, err := recompile(code)
	t.Logf("\n%s", src)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range interfaceDiffs(code, out) {
		t.Error(d)
	}
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}
