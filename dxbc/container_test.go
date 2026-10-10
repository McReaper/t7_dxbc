package dxbc

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/McReaper/t7_dxbc/internal/fixture"
)

func TestParse(t *testing.T) {
	b := fixture.Container(fixture.Chunk{FourCC: "RDEF", Data: []byte{1, 2, 3}}, fixture.Chunk{FourCC: "SHEX", Data: []byte{4, 5, 6, 7}}, fixture.Chunk{FourCC: "OSGN"})
	c, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Chunks) != 3 || !bytes.Equal(c.Chunk("SHDR", "SHEX"), []byte{4, 5, 6, 7}) || c.Chunk("ISGN") != nil ||
		len(c.Chunk("OSGN")) != 0 || c.Chunk("OSGN") == nil {
		t.Errorf("got %+v", c.Chunks)
	}
	// trailing bytes past the container's own size are not its data
	if c, err := Parse(append(b, 9, 9, 9)); err != nil || len(c.Chunks) != 3 {
		t.Errorf("trailing bytes: %v, %+v", err, c)
	}
}

func TestParseRejects(t *testing.T) {
	good := fixture.Container(fixture.Chunk{FourCC: "SHEX", Data: []byte{1, 2, 3, 4}})
	le := binary.LittleEndian
	cases := map[string]func(b []byte) []byte{
		"magic":                func(b []byte) []byte { copy(b, "DXBX"); return b },
		"short":                func(b []byte) []byte { return b[:20] },
		"total past the file":  func(b []byte) []byte { le.PutUint32(b[24:], uint32(len(b)+1)); return b },
		"total under a header": func(b []byte) []byte { le.PutUint32(b[24:], 8); return b },
		"chunk count":          func(b []byte) []byte { le.PutUint32(b[28:], 1<<30); return b },
		"chunk offset":         func(b []byte) []byte { le.PutUint32(b[headerLen:], uint32(len(b))); return b },
		"chunk size":           func(b []byte) []byte { le.PutUint32(b[headerLen+8:], 5); return b },
	}
	for name, corrupt := range cases {
		if _, err := Parse(corrupt(bytes.Clone(good))); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add(fixture.Container(fixture.Chunk{FourCC: "RDEF", Data: []byte{1, 2}}, fixture.Chunk{FourCC: "SHEX", Data: []byte{3}}))
	f.Add([]byte("DXBC"))
	f.Fuzz(func(t *testing.T, b []byte) {
		c, err := Parse(b)
		if err != nil {
			return
		}
		for _, ch := range c.Chunks {
			if len(ch.FourCC) != 4 {
				t.Fatalf("fourcc %q", ch.FourCC)
			}
		}
	})
}
