package dxbc

import (
	"os"
	"path/filepath"
	"testing"
)

// seedChunks adds every fixture's chunk of the given kinds to a fuzz corpus.
func seedChunks(f *testing.F, fourccs ...string) {
	paths, _ := filepath.Glob(filepath.Join("testdata", "*.dxbc"))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		c, err := Parse(b)
		if err != nil {
			f.Fatal(err)
		}
		for _, ch := range c.Chunks {
			for _, want := range fourccs {
				if ch.FourCC == want {
					f.Add(ch.Data)
				}
			}
		}
	}
}

// A corrupt RDEF is an error, never a panic or a runaway: types may point at
// each other, counts may claim more than the chunk holds.
func FuzzReadReflection(f *testing.F) {
	seedChunks(f, "RDEF")
	f.Fuzz(func(t *testing.T, b []byte) {
		ref, err := ReadReflection(b)
		if err != nil {
			return
		}
		for _, buf := range ref.Buffers {
			for _, v := range buf.Vars {
				_ = v.Type.HLSLName()
			}
		}
	})
}

func FuzzReadSignature(f *testing.F) {
	seedChunks(f, "ISGN", "OSGN")
	// a table whose offset plus a record's size wraps past zero: the element
	// was read in bounds and its stream field 4 GiB away
	f.Add(append([]byte{3, 0, 0, 0, 0xfc, 0xff, 0xff, 0xff}, make([]byte, 76)...))
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, fourcc := range []string{"ISGN", "OSG5", "ISG1"} {
			_, _ = ReadSignature(fourcc, b)
		}
	})
}
