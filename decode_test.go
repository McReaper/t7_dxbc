package t7dxbc

import (
	"encoding/binary"
	"testing"
)

// program lays out a SHEX chunk: a pixel shader 5.0 version token, the length
// token, then the instruction tokens.
func program(tokens ...uint32) []byte {
	b := binary.LittleEndian.AppendUint32(nil, 0x50)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(tokens)+2))
	for _, t := range tokens {
		b = binary.LittleEndian.AppendUint32(b, t)
	}
	return b
}

func TestDecodeProgram(t *testing.T) {
	// mov r0.xyzw, l(1.0, 0, 0, 0); ret
	mov := uint32(OpMov) | 8<<24
	dst := uint32(2) | 0<<2 | 0xF<<4 | uint32(OperandTemp)<<12 | 1<<20
	imm := uint32(2) | 1<<2 | 0xE4<<4 | uint32(OperandImmediate32)<<12
	p, err := DecodeProgram(program(mov, dst, 0, imm, 0x3F800000, 0, 0, 0, uint32(OpRet)|1<<24))
	if err != nil {
		t.Fatal(err)
	}
	got := p.Disassemble()
	if p.Stage != StagePixel || p.Major != 5 || len(got) != 2 || got[0] != "mov r0.xyzw, l(1.000000,0,0,0)" || got[1] != "ret" {
		t.Errorf("stage %d %d.%d: %q", p.Stage, p.Major, p.Minor, got)
	}
}

func TestDecodeProgramRejects(t *testing.T) {
	cases := map[string][]byte{
		"short chunk":             {1, 2, 3},
		"unaligned":               append(program(uint32(OpRet)|1<<24), 0),
		"length past the chunk":   func() []byte { b := program(uint32(OpRet) | 1<<24); b[4] = 9; return b }(),
		"zero-length opcode":      program(uint32(OpRet)),
		"opcode past the end":     program(uint32(OpMov) | 4<<24),
		"unknown opcode":          program(0x7FF | 1<<24),
		"operand past the end":    program(uint32(OpMov)|2<<24, 2|0xF<<4|1<<20),
		"N-component operand":     program(uint32(OpMov)|2<<24, 3),
		"index representation":    program(uint32(OpMov)|3<<24, 2|0xF<<4|1<<20|7<<22, 0),
		"customdata past the end": program(uint32(OpCustomdata), 40),
	}
	for name, b := range cases {
		if _, err := DecodeProgram(b); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// A corrupt program is an error, never a panic or a hang; whatever decodes
// also disassembles.
func FuzzDecodeProgram(f *testing.F) {
	seedChunks(f, "SHEX", "SHDR")
	f.Add(program(uint32(OpRet) | 1<<24))
	f.Fuzz(func(t *testing.T, b []byte) {
		p, err := DecodeProgram(b)
		if err != nil {
			return
		}
		_ = p.Disassemble()
	})
}
