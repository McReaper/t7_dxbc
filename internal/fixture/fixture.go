// Package fixture serves the tests the compiled shaders under the module's
// testdata, wherever a test runs from, and lays out DXBC containers.
package fixture

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Dir is the module's testdata directory.
func Dir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

// Shaders are the compiled shaders under testdata, by file name, each with
// Microsoft's disassembly beside it.
func Shaders(t testing.TB) map[string][2][]byte {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(Dir(), "*.dxbc"))
	if len(paths) == 0 {
		t.Fatal("no fixtures: go run ./internal/genfixtures")
	}
	out := map[string][2][]byte{}
	for _, p := range paths {
		code, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		asm, err := os.ReadFile(strings.TrimSuffix(p, ".dxbc") + ".asm")
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.Base(p)] = [2][]byte{code, asm}
	}
	return out
}

// Chunk is a container chunk, as dxbc.Chunk.
type Chunk struct {
	FourCC string
	Data   []byte
}

// Container lays out a DXBC file holding the given chunks.
func Container(chunks ...Chunk) []byte {
	const headerLen = 32 // magic, 16-byte checksum, version, total size, chunk count
	le := binary.LittleEndian
	b := make([]byte, headerLen+4*len(chunks))
	copy(b, "DXBC")
	le.PutUint32(b[20:], 1)
	le.PutUint32(b[28:], uint32(len(chunks)))
	for i, c := range chunks {
		le.PutUint32(b[headerLen+4*i:], uint32(len(b)))
		b = append(b, c.FourCC...)
		b = le.AppendUint32(b, uint32(len(c.Data)))
		b = append(b, c.Data...)
	}
	le.PutUint32(b[24:], uint32(len(b)))
	return b
}
