package t7dxbc

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var le = binary.LittleEndian

// Chunk is one part of a container, named by its four-character code.
type Chunk struct {
	FourCC string
	Data   []byte
}

// Container is a parsed DXBC file.
type Container struct {
	Chunks []Chunk
}

const headerLen = 32 // magic, 16-byte checksum, version, total size, chunk count

// Parse reads a DXBC container. It doesn't verify the checksum: the data the
// chunks hold is checked as each is read.
func Parse(b []byte) (*Container, error) {
	if len(b) < headerLen || string(b[:4]) != "DXBC" {
		return nil, errors.New("not a DXBC container")
	}
	total, n := le.Uint32(b[24:]), le.Uint32(b[28:])
	if uint64(total) > uint64(len(b)) || total < headerLen {
		return nil, fmt.Errorf("container claims %d bytes, file has %d", total, len(b))
	}
	b = b[:total]
	if uint64(n) > uint64(len(b)-headerLen)/4 {
		return nil, fmt.Errorf("%d chunks don't fit the container", n)
	}
	c := &Container{Chunks: make([]Chunk, 0, n)}
	for i := range n {
		off := le.Uint32(b[headerLen+4*i:])
		if uint64(off)+8 > uint64(len(b)) {
			return nil, fmt.Errorf("chunk %d starts past the container", i)
		}
		size := le.Uint32(b[off+4:])
		if uint64(off)+8+uint64(size) > uint64(len(b)) {
			return nil, fmt.Errorf("chunk %d (%q) runs past the container", i, b[off:off+4])
		}
		c.Chunks = append(c.Chunks, Chunk{FourCC: string(b[off : off+4]), Data: b[off+8 : off+8+size]})
	}
	return c, nil
}

// Chunk returns the data of the first chunk named by any of fourccs, in the
// order given, or nil.
func (c *Container) Chunk(fourccs ...string) []byte {
	for _, f := range fourccs {
		for _, ch := range c.Chunks {
			if ch.FourCC == f {
				return ch.Data
			}
		}
	}
	return nil
}
