package hlsl

import (
	"fmt"
	"strings"

	"github.com/McReaper/t7dxbc/dxbc"
)

// A geometry shader takes its primitive's vertices as an array of a struct,
// and appends the vertices it makes, each a struct, to a stream. Its inputs
// v[vertex][register] read the struct's fields; its outputs o# write the
// fields of the vertex being built, which emit appends and cut ends a strip
// after.

// geometry is what a geometry shader declares.
type geometry struct {
	primitive, topology uint32
	maxVertices         uint32
	instances           uint32
	streams             []uint32
}

// primitiveInputs are each input primitive's HLSL name and vertex count.
var primitiveInputs = map[uint32]struct {
	name     string
	vertices int
}{1: {"point", 1}, 2: {"line", 2}, 3: {"triangle", 3}, 6: {"lineadj", 4}, 7: {"triangleadj", 6}}

// streamTypes are the stream object of each output topology.
var streamTypes = map[uint32]string{1: "PointStream", 3: "LineStream", 5: "TriangleStream"}

func init() {
	declarations[dxbc.OpDclGsInputPrimitive] = func(e *emitter, in *dxbc.Instruction) { e.gs.primitive = in.Primitive() }
	declarations[dxbc.OpDclGsOutputPrimitiveTopology] = func(e *emitter, in *dxbc.Instruction) {
		e.gs.topology = in.Primitive()
	}
	declarations[dxbc.OpDclMaxOutputVertexCount] = func(e *emitter, in *dxbc.Instruction) {
		if len(in.Payload) > 0 {
			e.gs.maxVertices = in.Payload[0]
		}
	}
	declarations[dxbc.OpDclGsInstanceCount] = func(e *emitter, in *dxbc.Instruction) {
		if len(in.Payload) > 0 {
			e.gs.instances = in.Payload[0]
		}
	}
	declarations[dxbc.OpDclStream] = func(e *emitter, in *dxbc.Instruction) {
		if m, ok := operandRegister(in); ok {
			e.gs.streams = append(e.gs.streams, m)
		}
	}
	for _, op := range []dxbc.Opcode{dxbc.OpEmit, dxbc.OpEmitStream, dxbc.OpCut, dxbc.OpCutStream, dxbc.OpEmitthencut, dxbc.OpEmitthencutStream} {
		otherStatements[op] = func(e *emitter, in *dxbc.Instruction, _ bool) { e.emitStatement(in) }
	}
}

// fieldName is a signature element's field in a geometry shader's vertex
// structs.
func fieldName(el dxbc.Element) string { return fmt.Sprintf("%s%d", identifier(el.Name), el.Index) }

// inputName is what reads an input element: its parameter, or for a
// geometry shader the field of the vertex the index picks.
func (e *emitter) inputName(el dxbc.Element, vertex string) string {
	field := e.inFields[slot{el.Register, el.Mask}]
	if e.p.Stage == dxbc.StageGeometry {
		return fmt.Sprintf("in_v[%s].%s", vertex, field)
	}
	return "in_" + field
}

// outputName is what writes an output element.
func (e *emitter) outputName(el dxbc.Element) string {
	field := e.outFields[slot{el.Register, el.Mask}]
	if e.p.Stage == dxbc.StageGeometry {
		return "out_v." + field
	}
	return "out_" + field
}

// checkGeometry rejects what the decompiler doesn't write yet: several
// streams, an output on a stream other than 0.
func (e *emitter) checkGeometry() error {
	if len(e.gs.streams) > 1 {
		return fmt.Errorf("%d output streams: only one is decompiled", len(e.gs.streams))
	}
	for _, el := range e.out {
		if el.Stream != 0 {
			return fmt.Errorf("output %s%d on stream %d: only stream 0 is decompiled", el.Name, el.Index, el.Stream)
		}
	}
	if _, ok := primitiveInputs[e.gs.primitive]; !ok {
		return fmt.Errorf("input primitive %d", e.gs.primitive)
	}
	if _, ok := streamTypes[e.gs.topology]; !ok {
		return fmt.Errorf("output topology %d", e.gs.topology)
	}
	return nil
}

// vertexStructs writes the input and output vertex structs, their fields
// with the interpolation modes that pack them as the signatures do.
func (e *emitter) vertexStructs() {
	for _, s := range []struct {
		name  string
		decls []sigDecl
	}{{"GSIn", e.inDecls}, {"GSOut", e.outDecls}} {
		e.line("struct %s", s.name)
		e.line("{")
		for _, d := range s.decls {
			e.line("  %s;", d.declaration(""))
		}
		e.line("};")
		e.line("")
	}
}

// geometryEntry writes a geometry shader's attributes and parameters.
func (e *emitter) geometryEntry() {
	e.line("[maxvertexcount(%d)]", e.gs.maxVertices)
	if e.gs.instances > 1 {
		e.line("[instance(%d)]", e.gs.instances)
	}
	prim := primitiveInputs[e.gs.primitive]
	params := []string{
		fmt.Sprintf("%s GSIn in_v[%d]", prim.name, prim.vertices),
		fmt.Sprintf("inout %s<GSOut> out_stream", streamTypes[e.gs.topology]),
	}
	used := e.usedOperands()
	for _, t := range sortedTypes(systemInputs) {
		if used[t] {
			params = append(params, fmt.Sprintf("%s %s : %s", systemInputs[t][0], t.Prefix(), systemInputs[t][1]))
		}
	}
	e.line("void %s(%s)", Entry(e.p.Stage), strings.Join(params, ",\n  "))
	e.line("{")
	e.line("  GSOut out_v = (GSOut)0;")
}

// emitStatement writes emit and cut: append the vertex built, end the
// strip.
func (e *emitter) emitStatement(in *dxbc.Instruction) {
	switch in.Op {
	case dxbc.OpEmit, dxbc.OpEmitStream:
		e.line("out_stream.Append(out_v);")
	case dxbc.OpCut, dxbc.OpCutStream:
		e.line("out_stream.RestartStrip();")
	default:
		e.line("out_stream.Append(out_v);")
		e.line("out_stream.RestartStrip();")
	}
}
