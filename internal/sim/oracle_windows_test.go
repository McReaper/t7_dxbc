//go:build windows

package sim

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/McReaper/dxbc"
	"github.com/McReaper/dxbc/internal/d3dc"
)

// runs is how many random inputs each shader is checked on.
const runs = 12

// shader is a compiled shader, read.
type shader struct {
	p       *dxbc.Program
	ref     *dxbc.Reflection
	in, out []dxbc.Element
	m       *Machine
}

func readShader(code []byte) (*shader, error) {
	c, err := dxbc.Parse(code)
	if err != nil {
		return nil, err
	}
	p, err := dxbc.DecodeProgram(c.Chunk("SHEX", "SHDR"))
	if err != nil {
		return nil, err
	}
	ref := &dxbc.Reflection{}
	if rdef := c.Chunk("RDEF"); rdef != nil {
		if ref, err = dxbc.ReadReflection(rdef); err != nil {
			return nil, err
		}
	}
	in, out, err := c.Signatures()
	if err != nil {
		return nil, err
	}
	m, err := New(p, ref)
	if err != nil {
		return nil, err
	}
	return &shader{p, ref, in, out, m}, nil
}

var targets = map[dxbc.Stage]string{dxbc.StagePixel: "ps", dxbc.StageVertex: "vs", dxbc.StageCompute: "cs"}

// recompile decompiles a shader and compiles the result again.
func recompile(code []byte) (orig, again *shader, src string, err error) {
	if orig, err = readShader(code); err != nil {
		return nil, nil, "", fmt.Errorf("read: %w", err)
	}
	if src, err = dxbc.Decompile(code); err != nil {
		return orig, nil, src, fmt.Errorf("decompile: %w", err)
	}
	target := fmt.Sprintf("%s_%d_%d", targets[orig.p.Stage], orig.p.Major, orig.p.Minor)
	out, err := d3dc.Compile([]byte(src), "decompiled.hlsl", "main", target)
	if err != nil {
		return orig, nil, src, fmt.Errorf("compile: %w", err)
	}
	if again, err = readShader(out); err != nil {
		return orig, nil, src, fmt.Errorf("read recompiled: %w", err)
	}
	return orig, again, src, nil
}

// near compares two results' dwords: floats (normal, or zero) within a
// relative tolerance, anything else exactly.
func near(a, b uint32) bool {
	if a == b {
		return true
	}
	floatLike := func(x uint32) bool {
		e := x >> 23 & 0xFF
		return x&^(1<<31) == 0 || e != 0 && e != 0xFF
	}
	if !floatLike(a) || !floatLike(b) {
		return false
	}
	fa, fb := float64(math.Float32frombits(a)), float64(math.Float32frombits(b))
	return math.Abs(fa-fb) <= 2e-3*math.Max(1, math.Max(math.Abs(fa), math.Abs(fb)))
}

// outcome is what one run of a shader leaves: its outputs by element
// component, and the memory it wrote.
type outcome struct {
	values    map[string]uint32
	discarded bool
}

func (s *shader) run(seed uint32) (outcome, error) {
	w := NewWorld(s.ref, seed)
	o := outcome{values: map[string]uint32{}}
	if s.p.Stage == dxbc.StageCompute {
		if err := s.m.RunGroup(w, [3]uint32{seed % 3, seed % 2, 0}); err != nil {
			return o, err
		}
	} else {
		r, err := s.m.Run(w, Inputs(s.in, seed), nil)
		if err != nil {
			return o, err
		}
		o.discarded = r.Discarded
		for _, el := range s.out {
			if el.Register == dxbc.NoRegister {
				continue
			}
			for c := range 4 {
				if el.Mask&(1<<c) != 0 {
					o.values[fmt.Sprintf("%s%d.%c", el.Name, el.Index, "xyzw"[c])] = r.Outputs[el.Register][c]
				}
			}
		}
		for t, v := range r.System {
			o.values[fmt.Sprintf("system %d", t)] = v
		}
	}
	for c, v := range w.UAV {
		o.values[fmt.Sprintf("u%d[%d,%d,%d].%d", c.Reg, c.A, c.B, c.C, c.Comp)] = v
	}
	return o, nil
}

// differences compares two runs and names the first values that differ.
func differences(a, b outcome) []string {
	if a.discarded || b.discarded {
		if a.discarded != b.discarded {
			return []string{fmt.Sprintf("discarded %v, recompiled %v", a.discarded, b.discarded)}
		}
		return nil
	}
	keys := map[string]bool{}
	for k := range a.values {
		keys[k] = true
	}
	for k := range b.values {
		keys[k] = true
	}
	var diffs []string
	for k := range keys {
		va, oka := a.values[k]
		vb, okb := b.values[k]
		if oka != okb || !near(va, vb) {
			diffs = append(diffs, fmt.Sprintf("%s: %#x (%g) vs %#x (%g)", k, va, math.Float32frombits(va), vb,
				math.Float32frombits(vb)))
		}
	}
	sort.Strings(diffs)
	return diffs
}

// agreement runs both shaders on the same random inputs: how many runs
// differ, and the first difference.
func agreement(orig, again *shader) (failed int, first string, err error) {
	compared := 0
	for seed := uint32(1); seed <= runs; seed++ {
		a, errA := orig.run(seed)
		b, errB := again.run(seed)
		switch {
		case errors.Is(errA, ErrSteps) && errors.Is(errB, ErrSteps):
			continue
		case errA != nil:
			return 0, "", fmt.Errorf("original: %w", errA)
		case errB != nil:
			return 0, "", fmt.Errorf("recompiled: %w", errB)
		}
		compared++
		if d := differences(a, b); len(d) > 0 {
			failed++
			if first == "" {
				first = fmt.Sprintf("seed %d: %s", seed, strings.Join(d[:min(4, len(d))], "; "))
			}
		}
	}
	if compared == 0 {
		return 0, "", ErrSteps
	}
	return failed, first, nil
}

func TestFixturesAgree(t *testing.T) {
	if !d3dc.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	paths, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "*.dxbc"))
	if len(paths) == 0 {
		t.Fatal("no fixtures")
	}
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			code, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			orig, again, src, err := recompile(code)
			if err != nil {
				t.Fatalf("%v\n%s", err, src)
			}
			failed, first, err := agreement(orig, again)
			if err != nil {
				t.Fatal(err)
			}
			if failed > 0 {
				t.Errorf("%d of %d runs differ; %s\n%s", failed, runs, first, src)
			}
		})
	}
}

// TestCorpusAgrees is the opt-in semantic oracle over a shader cache
// ($DXBC_CORPUS, sampled by $DXBC_CORPUS_LIMIT): each shader that recompiles
// runs on random inputs beside the original. With $DXBC_JOURNAL each result
// is appended to that file as it comes, and a rerun skips the shaders it
// already holds: a run over a whole cache takes hours.
func TestCorpusAgrees(t *testing.T) {
	dir := os.Getenv("DXBC_CORPUS")
	if dir == "" || !d3dc.Available() {
		t.Skip("set DXBC_CORPUS to a directory of compiled shaders")
	}
	j, err := openJournal(os.Getenv("DXBC_JOURNAL"))
	if err != nil {
		t.Fatal(err)
	}
	defer j.close()
	var todo []string
	for _, p := range corpusPaths(t, dir) {
		if _, ok := j.done[filepath.Base(p)]; !ok {
			todo = append(todo, p)
		}
	}
	// the largest first: they take longest to compile
	sort.SliceStable(todo, func(a, b int) bool { return size(todo[a]) > size(todo[b]) })
	var wg sync.WaitGroup
	next := make(chan string)
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range next {
				kind, detail := checkOne(p)
				j.add(journalEntry{filepath.Base(p), kind, detail})
			}
		}()
	}
	for _, p := range todo {
		next <- p
	}
	close(next)
	wg.Wait()
	summarize(t, j.done)
}

// journalEntry is one shader's result.
type journalEntry struct{ name, kind, detail string }

// journal keeps results in a file, a line each, tab-separated.
type journal struct {
	mu   sync.Mutex
	f    *os.File
	done map[string]journalEntry
}

func openJournal(path string) (*journal, error) {
	j := &journal{done: map[string]journalEntry{}}
	if path == "" {
		return j, nil
	}
	if b, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if parts := strings.SplitN(line, "\t", 3); len(parts) == 3 {
				j.done[parts[0]] = journalEntry{parts[0], parts[1], parts[2]}
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	j.f = f
	return j, nil
}

func (j *journal) add(e journalEntry) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.done[e.name] = e
	if j.f != nil {
		detail := strings.NewReplacer("\t", " ", "\n", " ").Replace(e.detail)
		fmt.Fprintf(j.f, "%s\t%s\t%s\n", e.name, e.kind, detail)
	}
}

func (j *journal) close() {
	if j.f != nil {
		_ = j.f.Close()
	}
}

// summarize logs how many shaders fell in each kind, with the smallest as an
// example.
func summarize(t *testing.T, done map[string]journalEntry) {
	counts := map[string]int{}
	example := map[string]journalEntry{}
	dir := os.Getenv("DXBC_CORPUS")
	for _, e := range done {
		counts[e.kind]++
		ex, ok := example[e.kind]
		if !ok || size(filepath.Join(dir, e.name)) < size(filepath.Join(dir, ex.name)) {
			example[e.kind] = e
		}
	}
	kinds := make([]string, 0, len(counts))
	for k := range counts {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(a, b int) bool { return counts[kinds[a]] > counts[kinds[b]] })
	for _, k := range kinds {
		t.Logf("%5d  %s  (e.g. %s: %s)", counts[k], k, example[k].name, example[k].detail)
	}
}

// checkOne classifies one corpus file: "agrees", "differs", or why it
// couldn't be checked.
func checkOne(path string) (kind, detail string) {
	code, err := os.ReadFile(path)
	if err != nil || len(code) < 4 || string(code[:4]) != "DXBC" {
		return "not a shader", ""
	}
	orig, again, _, err := recompile(code)
	if err != nil {
		msg, _, _ := strings.Cut(err.Error(), "\n")
		k, _, _ := strings.Cut(msg, ":")
		return "not recompiled: " + k, msg[:min(len(msg), 200)]
	}
	failed, first, err := agreement(orig, again)
	switch {
	case errors.Is(err, ErrSteps):
		return "too long to run", ""
	case err != nil:
		msg := err.Error()
		return "not simulated", msg[:min(len(msg), 200)]
	case failed == 0:
		return "agrees", ""
	case failed == runs:
		return "differs on every run", first[:min(len(first), 300)]
	}
	return "differs on some runs", first[:min(len(first), 300)]
}

func corpusPaths(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	if n, err := strconv.Atoi(os.Getenv("DXBC_CORPUS_LIMIT")); err == nil && n > 0 && n < len(paths) {
		step := len(paths) / n
		var sample []string
		for i := 0; i < len(paths) && len(sample) < n; i += step {
			sample = append(sample, paths[i])
		}
		paths = sample
	}
	return paths
}

func size(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return math.MaxInt64
	}
	return fi.Size()
}

// TestAgreeOne checks one shader, DXBC_ONE=<file>, and shows its
// decompilation when it differs.
func TestAgreeOne(t *testing.T) {
	p := os.Getenv("DXBC_ONE")
	if p == "" {
		t.Skip("set DXBC_ONE to a compiled shader")
	}
	code, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	orig, again, src, err := recompile(code)
	if err != nil {
		t.Fatalf("%v\n%s", err, src)
	}
	failed, first, err := agreement(orig, again)
	if err != nil {
		t.Fatal(err)
	}
	if failed > 0 {
		t.Errorf("%d of %d runs differ; %s\n%s", failed, runs, first, src)
	}
}
