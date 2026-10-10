package dxbc

import (
	"fmt"
	"strings"
)

// The names below are the ones Microsoft's disassembler prints in its
// reflection header; the tests hold them to it.

// Kind names what a binding binds: cbuffer, tbuffer, texture, sampler or UAV.
func (b Binding) Kind() string {
	switch {
	case b.Input == InputCBuffer:
		return "cbuffer"
	case b.Input == InputTBuffer:
		return "tbuffer"
	case b.Input == InputSampler && b.Flags&BindComparisonSampler != 0:
		return "sampler_c"
	case b.Input == InputSampler:
		return "sampler"
	case b.Input.IsUAV():
		return "UAV"
	}
	return "texture"
}

// Format names what a binding returns: float4, struct, byte, or NA.
func (b Binding) Format() string {
	switch b.Input {
	case InputCBuffer, InputTBuffer, InputSampler:
		return "NA"
	case InputStructured, InputUAVStructured, InputUAVAppendStructured, InputUAVConsumeStructured,
		InputUAVStructuredWithCounter:
		return "struct"
	case InputByteAddress, InputUAVByteAddress:
		return "byte"
	}
	name := b.Return.String()
	if n := b.Components(); n > 1 {
		name += fmt.Sprint(n)
	}
	return name
}

var dimNames = map[Dimension]string{
	DimBuffer: "buf", DimTexture1D: "1d", DimTexture1DArray: "1darray", DimTexture2D: "2d",
	DimTexture2DArray: "2darray", DimTexture2DMS: "2dMS", DimTexture2DMSArray: "2dMSarray", DimTexture3D: "3d",
	DimTextureCube: "cube", DimTextureCubeArray: "cubearray",
}

// DimName names a binding's shape: 2d, cube, buf…, r/o for a read-only
// structured or byte-address buffer, r/w for a UAV one, NA for the rest.
func (b Binding) DimName() string {
	switch b.Input {
	case InputCBuffer, InputTBuffer, InputSampler:
		return "NA"
	case InputStructured, InputByteAddress:
		return "r/o"
	case InputUAVStructured, InputUAVByteAddress, InputUAVTyped:
		if b.Input != InputUAVTyped {
			return "r/w"
		}
	case InputUAVAppendStructured:
		return "append"
	case InputUAVConsumeStructured:
		return "consume"
	case InputUAVStructuredWithCounter:
		return "r/w+cnt"
	}
	n, ok := dimNames[b.Dim]
	if !ok {
		return "NA"
	}
	if (b.Dim == DimTexture2DMS || b.Dim == DimTexture2DMSArray) && b.NumSamples > 0 {
		n += fmt.Sprint(b.NumSamples) // 2dMS4: a multisampled view names its sample count
	}
	return n
}

// Register names the register a binding occupies: t0, s1, cb2, u3.
func (b Binding) Register() string { return fmt.Sprintf("%s%d", b.RegisterPrefix(), b.Point) }

// RegisterPrefix names the register file a binding is in: t, s, cb, u.
func (b Binding) RegisterPrefix() string {
	switch {
	case b.Input == InputCBuffer:
		return "cb"
	case b.Input == InputSampler:
		return "s"
	case b.Input.IsUAV():
		return "u"
	}
	return "t"
}

var systemValueNames = map[SystemValue]string{
	SVUndefined: "NONE", SVPosition: "POS", SVClipDistance: "CLIPDST", SVCullDistance: "CULLDST",
	SVRenderTargetArrayIndex: "RTINDEX", SVViewportArrayIndex: "VPINDEX", SVVertexID: "VERTID",
	SVPrimitiveID: "PRIMID", SVInstanceID: "INSTID", SVIsFrontFace: "FFACE", SVSampleIndex: "SAMPLE",
	SVTarget: "TARGET", SVDepth: "DEPTH", SVCoverage: "COVERAGE", SVDepthGreaterEqual: "DEPTHGE",
	SVDepthLessEqual: "DEPTHLE", SVStencilRef: "STENCILREF", SVInnerCoverage: "INNERCOV",
}

// pixelOutputs are the system values a pixel shader's output signature
// leaves undefined: they come from the semantic's name.
var pixelOutputs = map[string]SystemValue{
	"sv_target": SVTarget, "sv_depth": SVDepth, "sv_coverage": SVCoverage, "sv_depthgreaterequal": SVDepthGreaterEqual,
	"sv_depthlessequal": SVDepthLessEqual, "sv_stencilref": SVStencilRef,
}

// SystemValueOf is the element's system value, with a pixel shader output's
// (SV_Target, SV_Depth…) taken from its semantic name, where the chunk leaves
// it undefined.
func (e Element) SystemValueOf() SystemValue {
	if e.SystemValue == SVUndefined {
		if sv, ok := pixelOutputs[strings.ToLower(e.Name)]; ok {
			return sv
		}
	}
	return e.SystemValue
}

// String is the system value's short name, as the disassembler prints it.
func (s SystemValue) String() string {
	if n, ok := systemValueNames[s]; ok {
		return n
	}
	return fmt.Sprintf("SV%d", uint32(s))
}

// NoRegister is the register of a signature element the pipeline doesn't
// hold in a numbered register: a pixel shader's depth, coverage or stencil
// output.
const NoRegister = ^uint32(0)

// MaskString spells a component mask with its letters in place: "xy  ",
// " y w".
func MaskString(m uint8) string {
	var sb strings.Builder
	for i, c := range "xyzw" {
		if m&(1<<i) != 0 {
			sb.WriteRune(c)
		} else {
			sb.WriteByte(' ')
		}
	}
	return sb.String()
}
