package t7dxbc

import (
	"fmt"
	"sort"
	"strings"
)

// declarations writes the structs, constant buffers, resources, samplers,
// UAVs, the immediate constant buffer and the group-shared memory.
func (e *emitter) declarations() {
	e.line("// Decompiled by github.com/McReaper/t7dxbc")
	e.line("")
	e.structDecls()
	// in the reflection's order, which the compiler keeps from the source
	for i := range e.ref.Buffers {
		for r, l := range e.cbuffers {
			if l.buf == &e.ref.Buffers[i] {
				e.cbufferDecl(r, l)
			}
		}
	}
	for i := range e.ref.Bindings {
		e.resourceDecl(&e.ref.Bindings[i])
	}
	if len(e.icb) > 0 {
		var rows []string
		for i := 0; i+4 <= len(e.icb); i += 4 {
			rows = append(rows, fmt.Sprintf("uint4(0x%08x, 0x%08x, 0x%08x, 0x%08x)", e.icb[i], e.icb[i+1], e.icb[i+2], e.icb[i+3]))
		}
		e.line("static const uint4 icb[%d] = { %s };", len(rows), strings.Join(rows, ", "))
	}
	gs := make([]uint32, 0, len(e.tgsm))
	for g := range e.tgsm {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i] < gs[j] })
	for _, g := range gs {
		d := e.tgsm[g]
		if d[0] == 0 {
			e.line("groupshared uint g%d[%d];", g, d[1]/4)
		} else {
			e.line("groupshared uint g%d[%d][%d];", g, d[1], d[0]/4)
		}
	}
	e.line("")
}

// structName is a struct type's name: the reflection's, or the one
// nameStructs gave it.
func (e *emitter) structName(t *Type) string {
	if name, ok := e.structNames[t]; ok {
		return name
	}
	return identifier(t.Name)
}

// nameStructs names the struct types the reflection leaves unnamed (it
// records names from Shader Model 5 on) after the variable or buffer holding
// them, lights_t, so that two of them don't both declare a struct _.
func (e *emitter) nameStructs() {
	e.structNames = map[*Type]string{}
	taken := map[string]bool{}
	e.visitStructs(func(t *Type, holder string) {
		name := identifier(t.Name)
		if t.Name == "" {
			name = identifier(holder) + "_t"
			for i := 2; taken[name]; i++ {
				name = fmt.Sprintf("%s_t%d", identifier(holder), i)
			}
		}
		taken[name] = true
		e.structNames[t] = name
	})
}

// visitStructs calls f once on every struct type the buffers use, members
// before the struct holding them, with the name of what holds it (a
// variable, a buffer, a member path), in the reflection's order: the output
// must not depend on map order.
func (e *emitter) visitStructs(f func(t *Type, holder string)) {
	seen := map[*Type]bool{}
	var visit func(t *Type, holder string)
	visit = func(t *Type, holder string) {
		if t.Class != ClassStruct || seen[t] {
			return
		}
		seen[t] = true
		for _, m := range t.Members {
			visit(m.Type, holder+"_"+m.Name)
		}
		f(t, holder)
	}
	for i := range e.ref.Buffers {
		for _, l := range e.cbuffers {
			if l.buf == &e.ref.Buffers[i] {
				for _, v := range l.vars {
					visit(v.Type, v.Name)
				}
			}
		}
	}
	for _, b := range e.ref.Bindings {
		if t := e.structs[b.Name]; t != nil {
			visit(t, b.Name)
		}
	}
}

// structDecls writes every struct type the buffers use, members first, and
// one struct for the types the reflection records twice under one name.
func (e *emitter) structDecls() {
	e.nameStructs()
	done := map[string]bool{}
	e.visitStructs(func(t *Type, _ string) {
		name := e.structName(t)
		if done[name] {
			return
		}
		done[name] = true
		e.line("struct %s", name)
		e.line("{")
		for _, m := range t.Members {
			e.line("  %s %s%s;", e.declType(m.Type), identifier(m.Name), arraySuffix(m.Type))
		}
		e.line("};")
		e.line("")
	})
}

// cbufferDecl writes a constant buffer with each variable at its offset. The
// $Globals buffer holds the shader's global variables: written as globals,
// they get the same name back.
func (e *emitter) cbufferDecl(reg uint32, l *cbufferLayout) {
	if l.buf.Name == "$Globals" {
		for _, v := range l.vars {
			e.line("%s %s%s;", e.declType(v.Type), identifier(v.Name), arraySuffix(v.Type))
		}
		e.line("")
		return
	}
	e.line("cbuffer %s : register(b%d)", identifier(l.buf.Name), reg)
	e.line("{")
	for _, v := range l.vars {
		e.line("  %s %s%s : packoffset(%s);", e.declType(v.Type), identifier(v.Name), arraySuffix(v.Type), packOffset(v.Offset))
	}
	e.line("}")
	e.line("")
}

var textureTypes = map[Dimension]string{
	DimBuffer: "Buffer", DimTexture1D: "Texture1D", DimTexture1DArray: "Texture1DArray", DimTexture2D: "Texture2D",
	DimTexture2DArray: "Texture2DArray", DimTexture2DMS: "Texture2DMS", DimTexture2DMSArray: "Texture2DMSArray",
	DimTexture3D: "Texture3D", DimTextureCube: "TextureCube", DimTextureCubeArray: "TextureCubeArray",
}

// elementType is what a typed texture or buffer holds: float4, uint2,
// unorm float4.
func elementType(b *Binding) string {
	t := b.Return.String()
	if n := b.Components(); n > 1 {
		t += fmt.Sprint(n)
	}
	return t
}

func (e *emitter) resourceDecl(b *Binding) {
	name, reg := identifier(b.Name), b.Register()
	if b.Input.IsUAV() && e.uavFlags[b.Point]&UAVGloballyCoherent != 0 {
		e.sb.WriteString("globallycoherent ")
	}
	switch b.Input {
	case InputSampler:
		kind := "SamplerState"
		if b.Flags&BindComparisonSampler != 0 {
			kind = "SamplerComparisonState"
		}
		e.line("%s %s : register(%s);", kind, name, reg)
	case InputTexture:
		kind, ok := textureTypes[b.Dim]
		if !ok {
			e.line("// %s: unknown texture dimension %d", name, b.Dim)
			return
		}
		args := elementType(b)
		if (b.Dim == DimTexture2DMS || b.Dim == DimTexture2DMSArray) && b.NumSamples > 0 {
			args += fmt.Sprintf(", %d", b.NumSamples)
		}
		e.line("%s<%s> %s : register(%s);", kind, args, name, reg)
	case InputStructured, InputUAVStructured, InputUAVAppendStructured, InputUAVConsumeStructured,
		InputUAVStructuredWithCounter:
		elem := "uint"
		if t := e.structs[b.Name]; t != nil {
			elem = e.declType(t)
		}
		e.line("%s<%s> %s : register(%s);", structuredKinds[b.Input], elem, name, reg)
	case InputByteAddress:
		e.line("ByteAddressBuffer %s : register(%s);", name, reg)
	case InputUAVByteAddress:
		e.line("RWByteAddressBuffer %s : register(%s);", name, reg)
	case InputUAVTyped:
		kind, ok := textureTypes[b.Dim]
		if !ok {
			e.line("// %s: unknown UAV dimension %d", name, b.Dim)
			return
		}
		e.line("RW%s<%s> %s : register(%s);", kind, elementType(b), name, reg)
	}
}

var structuredKinds = map[InputType]string{
	InputStructured: "StructuredBuffer", InputUAVStructured: "RWStructuredBuffer",
	InputUAVAppendStructured: "AppendStructuredBuffer", InputUAVConsumeStructured: "ConsumeStructuredBuffer",
	InputUAVStructuredWithCounter: "RWStructuredBuffer",
}
