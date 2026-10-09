package dxbc

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden decompilations in testdata")

// TestDecompileGolden compares each fixture's decompilation with the one in
// testdata/<name>.decompiled.hlsl, so any change to the output shows, on any
// OS. The Windows oracles say whether the output is right; -update records
// it once they do. Any change fails it, a harmless one too, so mutation
// testing sets DXBC_NO_GOLDEN to be judged by the oracles alone.
func TestDecompileGolden(t *testing.T) {
	if os.Getenv("DXBC_NO_GOLDEN") != "" {
		t.Skip("DXBC_NO_GOLDEN: judged by the oracles alone, as mutation testing does")
	}
	for name, f := range fixtures(t) {
		t.Run(name, func(t *testing.T) {
			src, err := Decompile(f[0])
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", strings.TrimSuffix(name, ".dxbc")+".decompiled.hlsl")
			if *update {
				if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (go test -run TestDecompileGolden -update)", err)
			}
			if src != strings.ReplaceAll(string(want), "\r\n", "\n") {
				t.Errorf("decompilation changed; if it is right, go test -run TestDecompileGolden -update\n%s", firstDiff(string(want), src))
			}
		})
	}
}

// firstDiff shows the first line two texts differ at.
func firstDiff(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range max(len(w), len(g)) {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return "line " + strconv.Itoa(i+1) + ":\n  want " + a + "\n  got  " + b
		}
	}
	return ""
}

// TestDecompileIsDeterministic decompiles each fixture several times: the
// output must not depend on Go's map order.
func TestDecompileIsDeterministic(t *testing.T) {
	for name, f := range fixtures(t) {
		first, err := Decompile(f[0])
		if err != nil {
			t.Fatal(err)
		}
		for range 10 {
			if again, _ := Decompile(f[0]); again != first {
				t.Fatalf("%s: two decompilations differ\n%s", name, firstDiff(first, again))
			}
		}
	}
}
