package hlsl

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/McReaper/t7dxbc/dxbc"

	"github.com/McReaper/t7dxbc/internal/fixture"
)

// shaderOf builds a container holding a program of the given stage (5.0) and
// tokens, and any extra chunks.
func shaderOf(stage dxbc.Stage, tokens []uint32, extra ...dxbc.Chunk) []byte {
	b := binary.LittleEndian.AppendUint32(nil, uint32(stage)<<16|0x50)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(tokens)+2))
	for _, t := range tokens {
		b = binary.LittleEndian.AppendUint32(b, t)
	}
	cs := []fixture.Chunk{{FourCC: "SHEX", Data: b}}
	for _, c := range extra {
		cs = append(cs, fixture.Chunk(c))
	}
	return fixture.Container(cs...)
}

// Tokens for the declarations the limits concern.
func dclTemps(n uint32) []uint32 { return []uint32{uint32(dxbc.OpDclTemps) | 2<<24, n} }

func dclIndexable(x, size uint32) []uint32 {
	return []uint32{uint32(dxbc.OpDclIndexableTemp) | 4<<24, x, size, 4}
}

func dclTGSMRaw(g, bytes uint32) []uint32 {
	return []uint32{uint32(dxbc.OpDclThreadGroupSharedMemoryRaw) | 4<<24, uint32(dxbc.OperandTGSM)<<12 | 1<<20, g, bytes}
}

func dclTGSMStructured(g, stride, count uint32) []uint32 {
	return []uint32{uint32(dxbc.OpDclThreadGroupSharedMemoryStructured) | 5<<24, uint32(dxbc.OperandTGSM)<<12 | 1<<20, g, stride, count}
}

func dclGroup(x, y, z uint32) []uint32 {
	return []uint32{uint32(dxbc.OpDclThreadGroup) | 4<<24, x, y, z}
}

var ret = []uint32{uint32(dxbc.OpRet) | 1<<24}

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
		stage dxbc.Stage
		toks  []uint32
		err   string // "" for accepted
	}{
		{"temps at the limit", dxbc.StagePixel, tokens(dclTemps(4096), ret), ""},
		{"temps past", dxbc.StagePixel, tokens(dclTemps(4097), ret), "4097 temporaries"},
		{"indexable temps count", dxbc.StagePixel, tokens(dclTemps(4000), dclIndexable(0, 50), dclIndexable(1, 47), ret), "4097 temporaries"},
		{"indexable temps at the limit", dxbc.StagePixel, tokens(dclTemps(4000), dclIndexable(0, 50), dclIndexable(1, 46), ret), ""},
		{"raw shared memory at the limit", dxbc.StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMRaw(0, 16384), dclTGSMRaw(1, 16384), ret), ""},
		{"raw shared memory past", dxbc.StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMRaw(0, 16384), dclTGSMRaw(1, 16388), ret), "32772 bytes"},
		{"structured shared memory counts stride times count", dxbc.StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMStructured(0, 64, 513), ret), "32832 bytes"},
		{"structured shared memory at the limit", dxbc.StageCompute, tokens(dclGroup(1, 1, 1), dclTGSMStructured(0, 64, 256), dclTGSMRaw(1, 16384), ret), ""},
		{"threads at the limit", dxbc.StageCompute, tokens(dclGroup(16, 8, 8), ret), ""},
		{"threads past, by any dimension", dxbc.StageCompute, tokens(dclGroup(16, 8, 9), ret), "1152 threads"},
		{"threads past, by x", dxbc.StageCompute, tokens(dclGroup(1025, 1, 1), ret), "1025 threads"},
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
	label := []uint32{uint32(dxbc.OpLabel) | 3<<24, uint32(dxbc.OperandLabel)<<12 | 1<<20, 0}
	cases := map[string][]byte{
		"not a container":     []byte("not dxbc"),
		"no program":          fixture.Container(fixture.Chunk{FourCC: "RDEF"}),
		"a broken program":    fixture.Container(fixture.Chunk{FourCC: "SHEX", Data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}),
		"a broken reflection": shaderOf(dxbc.StagePixel, ret, dxbc.Chunk{FourCC: "RDEF", Data: []byte{1, 2, 3}}),
		"a broken signature":  shaderOf(dxbc.StagePixel, ret, dxbc.Chunk{FourCC: "ISGN", Data: []byte{1, 2, 3}}),
		"a hull shader":       shaderOf(dxbc.StageHull, ret),
		"a domain shader":     shaderOf(dxbc.StageDomain, ret),
		"a subroutine":        shaderOf(dxbc.StagePixel, tokens(label, ret)),
		"several gs streams": shaderOf(dxbc.StageGeometry, tokens(
			[]uint32{uint32(dxbc.OpDclStream) | 3<<24, 16<<12 | 1<<20, 0},
			[]uint32{uint32(dxbc.OpDclStream) | 3<<24, 16<<12 | 1<<20, 1}, ret)),
		"a gs without a primitive": shaderOf(dxbc.StageGeometry, ret),
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
		"dcl_temps":             {uint32(dxbc.OpDclTemps) | 1<<24},
		"dcl_indexableTemp":     {uint32(dxbc.OpDclIndexableTemp) | 3<<24, 0, 4},
		"dcl_tgsm_raw":          {uint32(dxbc.OpDclThreadGroupSharedMemoryRaw) | 3<<24, uint32(dxbc.OperandTGSM)<<12 | 1<<20, 0},
		"dcl_tgsm_structured":   {uint32(dxbc.OpDclThreadGroupSharedMemoryStructured) | 4<<24, uint32(dxbc.OperandTGSM)<<12 | 1<<20, 0, 4},
		"dcl_tgsm no operand":   {uint32(dxbc.OpDclThreadGroupSharedMemoryRaw) | 2<<24, 16},
		"dcl_thread_group":      {uint32(dxbc.OpDclThreadGroup) | 3<<24, 8, 8},
		"dcl_input_ps no index": {uint32(dxbc.OpDclInputPs) | 2<<11 | 2<<24, uint32(dxbc.OperandInput) << 12},
		"dcl_maxout":            {uint32(dxbc.OpDclMaxOutputVertexCount) | 1<<24},
		"dcl_gsinstances":       {uint32(dxbc.OpDclGsInstanceCount) | 1<<24},
		"customdata no data":    {uint32(dxbc.OpCustomdata) | dxbc.CustomdataICB<<11, 2},
	}
	for name, decl := range cases {
		for _, stage := range []dxbc.Stage{dxbc.StagePixel, dxbc.StageCompute, dxbc.StageGeometry} {
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
