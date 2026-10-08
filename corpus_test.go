//go:build windows

package dxbc

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/McReaper/dxbc/internal/d3dc"
)

// corpus is the opt-in real-data oracle: every DXBC file in $DXBC_CORPUS (a
// game's shader cache, say), up to $DXBC_CORPUS_LIMIT of them, each checked
// against Microsoft's disassembly of it.
func corpus(t *testing.T) []string {
	dir := os.Getenv("DXBC_CORPUS")
	if dir == "" {
		t.Skip("set DXBC_CORPUS to a directory of compiled shaders")
	}
	if !d3dc.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
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
		step := len(paths) / n // an even sample across the corpus, not its first n names
		var sample []string
		for i := 0; i < len(paths) && len(sample) < n; i += step {
			sample = append(sample, paths[i])
		}
		paths = sample
	}
	return paths
}

func TestCorpusReflection(t *testing.T) {
	paths := corpus(t)
	kinds := map[string]int{}
	var first []string
	failed := 0
	for _, p := range paths {
		code, err := os.ReadFile(p)
		if err != nil || len(code) < 4 || string(code[:4]) != "DXBC" {
			continue
		}
		asm, err := d3dc.Disassemble(code)
		if err != nil {
			continue // not something Microsoft's tool reads either
		}
		diffs := reflectionDiffs(code, asm)
		if len(diffs) == 0 {
			continue
		}
		failed++
		for _, d := range diffs {
			kind, _, _ := strings.Cut(d, ":")
			kinds[kind]++
			if len(first) < 8 {
				first = append(first, filepath.Base(p)+": "+d)
			}
		}
	}
	t.Logf("%d shaders, %d with a difference", len(paths), failed)
	for k, n := range kinds {
		t.Logf("  %s: %d", k, n)
	}
	for _, f := range first {
		t.Error(f)
	}
}

func TestCorpusDisassembly(t *testing.T) {
	paths := corpus(t)
	kinds := map[string]int{}
	examples := map[string]string{}
	failed, lines := 0, 0
	for _, p := range paths {
		code, err := os.ReadFile(p)
		if err != nil || len(code) < 4 || string(code[:4]) != "DXBC" {
			continue
		}
		asm, err := d3dc.Disassemble(code)
		if err != nil {
			continue
		}
		diffs, mnems := disasmDiffs(code, asm)
		if len(diffs) == 0 {
			continue
		}
		failed++
		lines += len(diffs)
		for i, m := range mnems {
			kinds[m]++
			if _, ok := examples[m]; !ok {
				examples[m] = filepath.Base(p) + ":\n  " + diffs[i]
			}
		}
	}
	t.Logf("%d shaders, %d with a difference (%d lines)", len(paths), failed, lines)
	keys := make([]string, 0, len(kinds))
	for k := range kinds {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return kinds[keys[i]] > kinds[keys[j]] })
	for i, k := range keys {
		if i == 25 {
			break
		}
		t.Errorf("%d× %s — e.g. %s", kinds[k], k, examples[k])
	}
}
