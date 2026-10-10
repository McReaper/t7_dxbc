package t7dxbc

import "fmt"

// Element is one entry of an input, output or patch-constant signature: the
// semantic a register carries between pipeline stages.
type Element struct {
	Name         string // the semantic, e.g. TEXCOORD or SV_Position
	Index        uint32 // the semantic index: TEXCOORD1 is TEXCOORD, 1
	SystemValue  SystemValue
	Type         ComponentType
	Register     uint32
	Mask         uint8 // the components the element holds, x=1 y=2 z=4 w=8
	RWMask       uint8 // inputs: the components the program reads; outputs: those it never writes
	Stream       uint32
	MinPrecision uint32
}

// signature chunk layouts: the element size and where its fields start.
var signatureLayouts = map[string]struct {
	size, base uint32
	stream     bool
	precision  bool
}{
	"ISGN": {size: 24},
	"OSGN": {size: 24},
	"PCSG": {size: 24},
	"OSG5": {size: 28, base: 4, stream: true},
	"ISG1": {size: 32, base: 4, stream: true, precision: true},
	"OSG1": {size: 32, base: 4, stream: true, precision: true},
	"PSG1": {size: 32, base: 4, stream: true, precision: true},
}

// ReadSignature reads a signature chunk whose four-character code is fourcc
// (ISGN, OSGN, OSG5, ISG1, OSG1, PCSG or PSG1).
func ReadSignature(fourcc string, b []byte) ([]Element, error) {
	layout, ok := signatureLayouts[fourcc]
	if !ok {
		return nil, fmt.Errorf("%q isn't a signature chunk", fourcc)
	}
	r := &rdef{b: b}
	h, err := r.words(0, 2)
	if err != nil {
		return nil, fmt.Errorf("%s header is truncated", fourcc)
	}
	n, at := h[0], h[1]
	if err := r.table(at, n, layout.size, fourcc+" elements"); err != nil {
		return nil, err
	}
	out := make([]Element, n)
	for i := range out {
		e, err := readElement(r, at+uint32(i)*layout.size, layout.base)
		if err != nil {
			return nil, fmt.Errorf("%s element %d: %w", fourcc, i, err)
		}
		if layout.stream {
			e.Stream = le.Uint32(b[at+uint32(i)*layout.size:])
		}
		if layout.precision {
			e.MinPrecision, _ = r.u32(at + uint32(i)*layout.size + 28)
		}
		out[i] = e
	}
	return out, nil
}

func readElement(r *rdef, off, base uint32) (Element, error) {
	w, err := r.words(off+base, 6)
	if err != nil {
		return Element{}, err
	}
	e := Element{Index: w[1], SystemValue: SystemValue(w[2]), Type: ComponentType(w[3]), Register: w[4],
		Mask: uint8(w[5]), RWMask: uint8(w[5] >> 8)}
	if e.Name, err = r.str(w[0]); err != nil {
		return Element{}, err
	}
	return e, nil
}

// Signatures are a shader's input and output signatures, from whichever
// chunk variant the compiler wrote.
func (c *Container) Signatures() (in, out []Element, err error) {
	for _, f := range []string{"ISG1", "ISGN"} {
		if b := c.Chunk(f); b != nil {
			if in, err = ReadSignature(f, b); err != nil {
				return nil, nil, err
			}
			break
		}
	}
	for _, f := range []string{"OSG1", "OSG5", "OSGN"} {
		if b := c.Chunk(f); b != nil {
			if out, err = ReadSignature(f, b); err != nil {
				return nil, nil, err
			}
			break
		}
	}
	return in, out, nil
}
