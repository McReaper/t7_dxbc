package dxbc

import (
	"fmt"
	"strings"
)

// declFormats write the declarations, whose controls and payloads each have
// their own syntax.
var declFormats = map[Opcode]func(*Instruction) string{}

func init() {
	declFormats[OpDclGlobalFlags] = disasmGlobalFlags
	declFormats[OpDclTemps] = func(in *Instruction) string { return "dcl_temps " + payload(in, 0) }
	declFormats[OpDclConstantBuffer] = disasmConstantBuffer
	declFormats[OpDclSampler] = disasmSampler
	declFormats[OpDclResource] = disasmResource
	declFormats[OpDclResourceStructured] = func(in *Instruction) string {
		return "dcl_resource_structured " + declOperand(in) + ", " + payload(in, 0)
	}
	declFormats[OpDclResourceRaw] = func(in *Instruction) string { return "dcl_resource_raw " + declOperand(in) }
	declFormats[OpDclInput] = func(in *Instruction) string { return "dcl_input " + declOperand(in) }
	declFormats[OpDclInputSgv] = func(in *Instruction) string { return "dcl_input_sgv " + declOperand(in) + ", " + sysName(in) }
	declFormats[OpDclInputSiv] = func(in *Instruction) string { return "dcl_input_siv " + declOperand(in) + ", " + sysName(in) }
	declFormats[OpDclInputPs] = func(in *Instruction) string { return "dcl_input_ps " + interpolation(in) + declOperand(in) }
	declFormats[OpDclInputPsSgv] = func(in *Instruction) string {
		return "dcl_input_ps_sgv " + interpolation(in) + declOperand(in) + ", " + sysName(in)
	}
	declFormats[OpDclInputPsSiv] = func(in *Instruction) string {
		return "dcl_input_ps_siv " + interpolation(in) + declOperand(in) + ", " + sysName(in)
	}
	declFormats[OpDclOutput] = func(in *Instruction) string { return "dcl_output " + declOperand(in) }
	declFormats[OpDclOutputSgv] = func(in *Instruction) string { return "dcl_output_sgv " + declOperand(in) + ", " + sysName(in) }
	declFormats[OpDclOutputSiv] = func(in *Instruction) string { return "dcl_output_siv " + declOperand(in) + ", " + sysName(in) }
	declFormats[OpDclIndexableTemp] = func(in *Instruction) string {
		return fmt.Sprintf("dcl_indexableTemp x%s[%s], %s", payload(in, 0), payload(in, 1), payload(in, 2))
	}
	declFormats[OpDclUnorderedAccessViewTyped] = func(in *Instruction) string {
		ret := ""
		if len(in.Payload) > 0 {
			ret = returnTypeList(in.Payload[0]) + " "
		}
		return "dcl_uav_typed_" + resourceDim(in.Token>>11&0x1F) + uavFlags(in) + " " + ret + declOperand(in)
	}
	declFormats[OpDclUnorderedAccessViewRaw] = func(in *Instruction) string {
		return "dcl_uav_raw" + uavFlags(in) + " " + declOperand(in)
	}
	declFormats[OpDclUnorderedAccessViewStructured] = func(in *Instruction) string {
		return "dcl_uav_structured" + uavFlags(in) + " " + declOperand(in) + ", " + payload(in, 0)
	}
	declFormats[OpDclThreadGroupSharedMemoryRaw] = func(in *Instruction) string {
		return "dcl_tgsm_raw " + declOperand(in) + ", " + payload(in, 0)
	}
	declFormats[OpDclThreadGroupSharedMemoryStructured] = func(in *Instruction) string {
		return "dcl_tgsm_structured " + declOperand(in) + ", " + payload(in, 0) + ", " + payload(in, 1)
	}
	declFormats[OpDclMaxOutputVertexCount] = func(in *Instruction) string { return "dcl_maxout " + payload(in, 0) }
	declFormats[OpDclGsOutputPrimitiveTopology] = func(in *Instruction) string {
		return "dcl_outputtopology " + named(topologies, in.Token>>11&0x3F)
	}
	declFormats[OpDclGsInputPrimitive] = func(in *Instruction) string {
		return "dcl_inputprimitive " + named(primitives, in.Token>>11&0x3F)
	}
	declFormats[OpDclThreadGroup] = func(in *Instruction) string {
		return fmt.Sprintf("dcl_thread_group %s, %s, %s", payload(in, 0), payload(in, 1), payload(in, 2))
	}
}

// uavFlags writes a UAV declaration's flags: _glc (globally coherent), _rov
// (rasterizer ordered), _opc (an order-preserving counter).
func uavFlags(in *Instruction) string {
	var s string
	for _, f := range []struct {
		bit  uint
		name string
	}{{16, "_glc"}, {17, "_rov"}, {23, "_opc"}} {
		if in.Token>>f.bit&1 != 0 {
			s += f.name
		}
	}
	return s
}

var topologies = map[uint32]string{1: "pointlist", 2: "linelist", 3: "linestrip", 4: "trianglelist", 5: "trianglestrip",
	10: "linelistadj", 11: "linestripadj", 12: "trianglelistadj", 13: "trianglestripadj"}

var primitives = map[uint32]string{1: "point", 2: "line", 3: "triangle", 6: "lineadj", 7: "triangleadj"}

func named(names map[uint32]string, v uint32) string {
	if n, ok := names[v]; ok {
		return n
	}
	return fmt.Sprint(v)
}

func payload(in *Instruction, i int) string {
	if i < len(in.Payload) {
		return fmt.Sprint(in.Payload[i])
	}
	return "?"
}

func declOperand(in *Instruction) string {
	if len(in.Operands) == 0 {
		return "?"
	}
	return formatOperand(&in.Operands[0], numTypeless)
}

var globalFlagNames = []string{"refactoringAllowed", "enableDoublePrecisionFloatOps", "forceEarlyDepthStencil",
	"enableRawAndStructuredBuffers", "skipOptimization", "enableMinimumPrecision", "enable11_1DoubleExtensions",
	"enable11_1ShaderExtensions"}

func disasmGlobalFlags(in *Instruction) string {
	var names []string
	flags := in.Token >> 11
	for i, n := range globalFlagNames {
		if flags&(1<<i) != 0 {
			names = append(names, n)
		}
	}
	return "dcl_globalFlags " + strings.Join(names, " | ")
}

func disasmConstantBuffer(in *Instruction) string {
	pattern := "immediateIndexed"
	if in.Token>>11&1 != 0 {
		pattern = "dynamicIndexed"
	}
	if len(in.Operands) == 0 || len(in.Operands[0].Index) < 2 {
		return "dcl_constantbuffer ?"
	}
	idx := in.Operands[0].Index
	return fmt.Sprintf("dcl_constantbuffer CB%d[%d], %s", idx[0].Imm, idx[1].Imm, pattern)
}

var samplerModes = []string{"mode_default", "mode_comparison", "mode_mono"}

func disasmSampler(in *Instruction) string {
	mode := in.Token >> 11 & 0xF
	name := fmt.Sprintf("mode%d", mode)
	if int(mode) < len(samplerModes) {
		name = samplerModes[mode]
	}
	return "dcl_sampler " + declOperand(in) + ", " + name
}

func disasmResource(in *Instruction) string {
	ret := ""
	if len(in.Payload) > 0 {
		ret = returnTypeList(in.Payload[0]) + " "
	}
	dim := resourceDim(in.Token >> 11 & 0x1F)
	if n := in.Token >> 16 & 0x7F; n != 0 { // a multisampled texture's sample count
		dim += fmt.Sprintf("(%d)", n)
	}
	return "dcl_resource_" + dim + " " + ret + declOperand(in)
}

var interpolationModes = []string{"", "constant", "linear", "linear centroid", "linear noperspective",
	"linear noperspective centroid", "linear sample", "linear noperspective sample"}

func interpolation(in *Instruction) string {
	mode := in.Token >> 11 & 0xF
	if int(mode) < len(interpolationModes) && mode != 0 {
		return interpolationModes[mode] + " "
	}
	return ""
}

var systemNames = map[uint32]string{
	1: "position", 2: "clip_distance", 3: "cull_distance", 4: "rendertarget_array_index", 5: "viewport_array_index",
	6: "vertex_id", 7: "primitive_id", 8: "instance_id", 9: "is_front_face", 10: "sampleIndex",
}

func sysName(in *Instruction) string {
	if len(in.Payload) == 0 {
		return "?"
	}
	if n, ok := systemNames[in.Payload[0]]; ok {
		return n
	}
	return fmt.Sprintf("name%d", in.Payload[0])
}

// disasmCustomdata writes an immediate constant buffer; other customdata
// blocks (comments, debug info) aren't instructions.
func disasmCustomdata(in *Instruction) string {
	if in.Token>>11 != 3 {
		return fmt.Sprintf("customdata %d", in.Token>>11)
	}
	var rows []string
	for i := 0; i+4 <= len(in.Payload); i += 4 {
		vals := make([]string, 4)
		for j := range vals {
			vals[j] = formatNumber(in.Payload[i+j], numTypeless)
		}
		rows = append(rows, "{ "+strings.Join(vals, ", ")+"}")
	}
	return "dcl_immediateConstantBuffer { " + strings.Join(rows, ", ") + " }"
}
