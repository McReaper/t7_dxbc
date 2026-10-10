package t7dxbc

import (
	"encoding/binary"
	"strings"
	"testing"
)

// shaderOf builds a container holding a program of the given stage (5.0) and
// tokens, and any extra chunks.
func shaderOf(stage Stage, tokens []uint32, extra ...Chunk) []byte {
	b := binary.LittleEndian.AppendUint32(nil, uint32(stage)<<16|0x50)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(tokens)+2))
	for _, t := range tokens {
		b = binary.LittleEndian.AppendUint32(b, t)
	}
	return container(append([]Chunk{{"SHEX", b}}, extra...)...)
}

// Tokens for the declarations the limits concern.
func dclTemps(n uint32) []uint32 { return []uint32{uint32(OpDclTemps) | 2<<24, n} }

func dclIndexable(x, size uint32) []uint32 {
	return []uint32{uint32(OpDclIndexableTemp) | 4<<24, x, size, 4}
}

func dclTGSMRaw(g, bytes uint32) []uint32 {
	return []uint32{uint32(OpDclThreadGroupSharedMemoryRaw) | 4<<24, uint32(OperandTGSM)<<12 | 1<<20, g, bytes}
}

func dclTGSMStructured(g, stride, count uint32) []uint32 {
	return []uint32{uint32(OpDclThreadGroupSharedMemoryStructured) | 5<<24, uint32(OperandTGSM)<<12 | 1<<20, g, stride, count}
}

func dclGroup(x, y, z uint32) []uint32 { return []uint32{uint32(OpDclThreadGroup) | 4<<24, x, y, z} }

var ret = []uint32{uint32(OpRet) | 1<<24}

func tokens(parts ...[]uint32) []uint32 {
	var out []uint32
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// Decompile accepts declarations up to Direct3D's limits and rejects them
// past: the limits are what keeps a hostile file from making huge HLSL.
func TestDecompileLimits(t *testing.T) {
	cases := []struct {
		name  string
		stage Stage
		toks  []uint32
		err   string // "" for accepted
	}{
		{"temps at the limit", StagePixel, tokens(dclTemps(4096), ret), ""},
		{"temps past", StagePixel, tokens(dclTemps(4097), ret), "4097 temporaries"},
		{"indexable temps count", StagePixel, tokens(dclTemps(4000), dclIndexable(0, 50), dclIndexable(1, 47), ret), "4097 temporaries"},
		{"indexable temps at the limit", StagePixel, tokens(dclTemps(4000), dclIndexable(0, 50), dclIndexable(1, 46), ret), ""},
		{"raw shared memory at the limit", StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMRaw(0, 16384), dclTGSMRaw(1, 16384), ret), ""},
		{"raw shared memory past", StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMRaw(0, 16384), dclTGSMRaw(1, 16388), ret), "32772 bytes"},
		{"structured shared memory counts stride times count", StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMStructured(0, 64, 513), ret), "32832 bytes"},
		{"structured shared memory at the limit", StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMStructured(0, 64, 256), dclTGSMRaw(1, 16384), ret), ""},
		{"threads at the limit", StageCompute, tokens(dclGroup(16, 8, 8), ret), ""},
		{"threads past, by any dimension", StageCompute, tokens(dclGroup(16, 8, 9), ret), "1152 threads"},
		{"threads past, by x", StageCompute, tokens(dclGroup(1025, 1, 1), ret), "1025 threads"},
	}
	for _, c := range cases {
		_, err := Decompile(shaderOf(c.stage, c.toks))
		switch {
		case c.err == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.err != "" && (err == nil || !strings.Contains(err.Error(), c.err)):
			t.Errorf("%s: error %v, want one saying %q", c.name, err, c.err)
		}
	}
}

// Decompile reports what it can't read or write as an error.
func TestDecompileRejects(t *testing.T) {
	label := []uint32{uint32(OpLabel) | 3<<24, uint32(OperandLabel)<<12 | 1<<20, 0}
	cases := map[string][]byte{
		"not a container":     []byte("not dxbc"),
		"no program":          container(Chunk{"RDEF", nil}),
		"a broken program":    container(Chunk{"SHEX", []byte{1, 2, 3, 4, 5, 6, 7, 8}}),
		"a broken reflection": shaderOf(StagePixel, ret, Chunk{"RDEF", []byte{1, 2, 3}}),
		"a broken signature":  shaderOf(StagePixel, ret, Chunk{"ISGN", []byte{1, 2, 3}}),
		"a hull shader":       shaderOf(StageHull, ret),
		"a domain shader":     shaderOf(StageDomain, ret),
		"a subroutine":        shaderOf(StagePixel, tokens(label, ret)),
		"several gs streams": shaderOf(StageGeometry, tokens(
			[]uint32{uint32(OpDclStream) | 3<<24, 16<<12 | 1<<20, 0},
			[]uint32{uint32(OpDclStream) | 3<<24, 16<<12 | 1<<20, 1}, ret)),
		"a gs without a primitive": shaderOf(StageGeometry, ret),
	}
	for name, b := range cases {
		if src, err := Decompile(b); err == nil {
			t.Errorf("%s: no error\n%s", name, src)
		}
	}
}

// Declarations missing their payload or operand are skipped, never read past
// their end.
func TestDecompileTruncatedDeclarations(t *testing.T) {
	cases := map[string][]uint32{
		"dcl_temps":             {uint32(OpDclTemps) | 1<<24},
		"dcl_indexableTemp":     {uint32(OpDclIndexableTemp) | 3<<24, 0, 4},
		"dcl_tgsm_raw":          {uint32(OpDclThreadGroupSharedMemoryRaw) | 3<<24, uint32(OperandTGSM)<<12 | 1<<20, 0},
		"dcl_tgsm_structured":   {uint32(OpDclThreadGroupSharedMemoryStructured) | 4<<24, uint32(OperandTGSM)<<12 | 1<<20, 0, 4},
		"dcl_tgsm no operand":   {uint32(OpDclThreadGroupSharedMemoryRaw) | 2<<24, 16},
		"dcl_thread_group":      {uint32(OpDclThreadGroup) | 3<<24, 8, 8},
		"dcl_input_ps no index": {uint32(OpDclInputPs) | 2<<11 | 2<<24, uint32(OperandInput) << 12},
		"dcl_maxout":            {uint32(OpDclMaxOutputVertexCount) | 1<<24},
		"dcl_gsinstances":       {uint32(OpDclGsInstanceCount) | 1<<24},
		"customdata no data":    {uint32(OpCustomdata) | CustomdataICB<<11, 2},
	}
	for name, decl := range cases {
		for _, stage := range []Stage{StagePixel, StageCompute, StageGeometry} {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("%s in stage %d: panic %v", name, stage, r)
					}
				}()
				_, _ = Decompile(shaderOf(stage, tokens(decl, ret)))
			}()
		}
	}
}
