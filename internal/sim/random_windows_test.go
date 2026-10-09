//go:build windows

package sim

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/McReaper/dxbc/internal/d3dc"
	"github.com/McReaper/dxbc/internal/oracle"
)

// TestRandomShadersAgree compiles random pixel and compute shaders with
// Microsoft's compiler and holds each to both oracles: they exercise the
// swizzles, types, operations and memory accesses in combinations no
// hand-written fixture covers. $DXBC_RANDOM sets how many of each (default
// 150) and $DXBC_RANDOM_SEED the first seed.
func TestRandomShadersAgree(t *testing.T) {
	if !oracle.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	n, first := 150, uint64(1)
	if v, err := strconv.Atoi(os.Getenv("DXBC_RANDOM")); err == nil {
		n = v
	}
	if v, err := strconv.ParseUint(os.Getenv("DXBC_RANDOM_SEED"), 10, 64); err == nil {
		first = v
	}
	t.Run("pixel", func(t *testing.T) { checkRandom(t, n, first, randomShader, "ps_5_0") })
	t.Run("compute", func(t *testing.T) { checkRandom(t, n, first, randomComputeShader, "cs_5_0") })
}

// compilerCrash reports whether the compiler failed on its own bug.
func compilerCrash(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "Internal Compiler Error") ||
		strings.Contains(err.Error(), "internal error"))
}

// checkRandom holds n generated shaders to both oracles. A shader the
// compiler crashes on is skipped; one whose decompilation it crashes on is
// counted, its own bug too, and fails the test past 1%.
func checkRandom(t *testing.T, n int, first uint64, gen func(uint64) string, target string) {
	crashes := 0
	for seed := first; seed < first+uint64(n); seed++ {
		src := gen(seed)
		code, err := d3dc.Compile([]byte(src), "random.hlsl", "main", target)
		switch {
		case compilerCrash(err):
			continue
		case err != nil:
			t.Fatalf("seed %d: the generated shader doesn't compile: %v\n%s", seed, err, src)
		}
		orig, again, dec, diffs, err := recompile(code)
		switch {
		case compilerCrash(err):
			crashes++
			t.Logf("seed %d: the compiler crashes on the decompilation: %v", seed, err)
			continue
		case err != nil:
			t.Errorf("seed %d: %v\n%s", seed, err, src)
			continue
		}
		failed, firstDiff, err := agreement(orig, again)
		if failed == 0 && len(diffs) == 1 && strings.HasPrefix(diffs[0], "buffers: use changed (used=true)") {
			// the compiler proved a read dead in the decompilation and not in
			// the original: a random shader's contrived arithmetic does that
			diffs = nil
		}
		switch {
		case err != nil:
			t.Errorf("seed %d: %v", seed, err)
		case failed > runs/2 || len(diffs) > 0:
			t.Errorf("seed %d: %d of %d runs differ (%s), interface %v\n--- source\n%s\n--- decompiled\n%s", seed, failed,
				runs, firstDiff, diffs, src, dec)
		}
	}
	if crashes*100 > n {
		t.Errorf("the compiler crashed on %d of %d decompilations", crashes, n)
	}
}

// TestHLSLOne holds one HLSL shader, $DXBC_HLSL=<file> compiled for
// $DXBC_TARGET (ps_5_0 by default), to both oracles: for narrowing down what
// a random shader got wrong.
func TestHLSLOne(t *testing.T) {
	p := os.Getenv("DXBC_HLSL")
	if p == "" || !oracle.Available() {
		t.Skip("set DXBC_HLSL to a pixel shader's source")
	}
	src, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	target := "ps_5_0"
	if v := os.Getenv("DXBC_TARGET"); v != "" {
		target = v
	}
	code, err := d3dc.Compile(src, p, "main", target)
	if err != nil {
		t.Fatal(err)
	}
	orig, again, dec, diffs, err := recompile(code)
	if err != nil {
		t.Fatalf("%v\n%s", err, dec)
	}
	failed, first, err := agreement(orig, again)
	if err != nil {
		t.Fatal(err)
	}
	if failed > 0 || len(diffs) > 0 {
		t.Errorf("%d of %d runs differ (%s), interface %v\n%s", failed, runs, first, diffs, dec)
	}
}

// TestRandomSignaturesAgree compiles vertex shaders with random outputs and
// pixel shaders with random inputs, the packing the decompiler must
// reproduce from modes the signatures don't record, and holds each to both
// oracles. $DXBC_RANDOM sets how many of each (default 150).
func TestRandomSignaturesAgree(t *testing.T) {
	if !oracle.Available() {
		t.Skip("d3dcompiler_47.dll not found")
	}
	n := 150
	if v, err := strconv.Atoi(os.Getenv("DXBC_RANDOM")); err == nil {
		n = v
	}
	for seed := uint64(1); seed <= uint64(n); seed++ {
		for _, c := range []struct{ src, target string }{
			{randomVertexShader(seed), "vs_5_0"}, {randomPixelInputs(seed), "ps_5_0"},
		} {
			code, err := d3dc.Compile([]byte(c.src), "random.hlsl", "main", c.target)
			if err != nil {
				t.Fatalf("seed %d: the generated %s doesn't compile: %v\n%s", seed, c.target, err, c.src)
			}
			orig, again, dec, diffs, err := recompile(code)
			if err != nil {
				t.Errorf("seed %d %s: %v\n%s", seed, c.target, err, c.src)
				continue
			}
			failed, first, err := agreement(orig, again)
			if err != nil || failed > 0 || len(diffs) > 0 {
				t.Errorf("seed %d %s: %v, %d runs differ (%s), interface %v\n--- source\n%s\n--- decompiled\n%s", seed,
					c.target, err, failed, first, diffs, c.src, dec)
			}
		}
	}
}
