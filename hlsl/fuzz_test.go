package hlsl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/McReaper/t7_dxbc/internal/fixture"
)

// FuzzDecompile feeds the decompiler whole containers, the fixtures and
// what the fuzzer makes of them: whatever the bytes, it returns HLSL or an
// error, never panics or hangs.
func FuzzDecompile(f *testing.F) {
	paths, _ := filepath.Glob(filepath.Join(fixture.Dir(), "*.dxbc"))
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		src, err := Decompile(b)
		if err == nil && src == "" {
			t.Fatal("no error and no source")
		}
	})
}
