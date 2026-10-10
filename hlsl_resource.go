package t7dxbc

import (
	"fmt"
	"strings"
)

// coords is how many address components a view of each dimension takes, and
// how many an immediate offset has (0: none allowed).
var coords = map[Dimension][2]int{
	DimBuffer: {1, 0}, DimTexture1D: {1, 1}, DimTexture1DArray: {2, 1}, DimTexture2D: {2, 2},
	DimTexture2DArray: {3, 2}, DimTexture2DMS: {2, 2}, DimTexture2DMSArray: {3, 2}, DimTexture3D: {3, 3},
	DimTextureCube: {3, 0}, DimTextureCubeArray: {4, 0}, DimBufferEx: {1, 0},
}

// binding finds the declaration of a t#, s# or u# operand: its HLSL name,
// an element of an array of resources by index.
func (e *emitter) binding(op *Operand) (string, *Binding) {
	if len(op.Index) == 0 || op.Index[0].Relative != nil {
		return "", nil
	}
	prefix := operandPrefixes[op.Type]
	reg := uint32(op.Index[0].Imm)
	for i := range e.ref.Bindings {
		b := &e.ref.Bindings[i]
		if b.registerPrefix() != prefix {
			continue
		}
		if reg < b.Point || reg >= b.Point+max(b.Count, 1) {
			continue
		}
		name := identifier(b.Name)
		if b.Count > 1 {
			name = fmt.Sprintf("%s[%d]", name, reg-b.Point)
		}
		return name, b
	}
	return "", nil
}

// resourceName is a resource operand's HLSL name, or "" with the operand
// recorded as unknown.
func (e *emitter) resourceName(op *Operand) (string, *Binding) {
	name, b := e.binding(op)
	if b == nil {
		e.unknown["resource "+formatOperand(op, numTypeless)+" without reflection"] = true
	}
	return name, b
}

// offsetArg is the immediate texel offset of a sample or load, ", int2(1, -1)",
// or "".
func offsetArg(in *Instruction, n int) string {
	o := in.Offsets()
	if n == 0 || o == [3]int32{} {
		return ""
	}
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprint(o[i])
	}
	if n == 1 {
		return ", " + parts[0]
	}
	return fmt.Sprintf(", int%d(%s)", n, strings.Join(parts, ", "))
}

// sampleForms are the sample instructions: the method, and the extra
// operands after the sampler it passes (the comparison value, the bias, the
// level, the gradients).
var sampleForms = map[Opcode]struct {
	method string
	extra  int
	scalar bool // returns one value: a comparison
}{
	OpSample: {"Sample", 0, false}, OpSampleB: {"SampleBias", 1, false}, OpSampleL: {"SampleLevel", 1, false},
	OpSampleD: {"SampleGrad", 2, false}, OpSampleC: {"SampleCmp", 1, true}, OpSampleCLz: {"SampleCmpLevelZero", 1, true},
}

// sampleStatement writes sample, sample_b, sample_l, sample_d, sample_c and
// sample_c_lz: dest, address, resource, sampler[, extras].
func (e *emitter) sampleStatement(in *Instruction, sat bool) {
	f := sampleForms[in.Op]
	res, b := e.resourceName(&in.Operands[2])
	smp, _ := e.resourceName(&in.Operands[3])
	if b == nil || smp == "" {
		return
	}
	n := coords[b.Dim]
	args := []string{smp, e.read(&in.Operands[1], first(&in.Operands[1], n[0]), tFloat)}
	for i := 4; i < 4+f.extra && i < len(in.Operands); i++ {
		cs := picks(&in.Operands[i], 1)
		if in.Op == OpSampleD {
			cs = first(&in.Operands[i], max(n[1], 1))
			if b.Dim == DimTextureCube || b.Dim == DimTextureCubeArray {
				cs = first(&in.Operands[i], 3)
			}
		}
		args = append(args, e.read(&in.Operands[i], cs, tFloat))
	}
	call := fmt.Sprintf("%s.%s(%s%s)", res, f.method, strings.Join(args, ", "), offsetArg(in, n[1]))
	e.writeResult(in, call, f.scalar, b, sat)
}

// writeResult assigns what a resource method returns, through the resource
// operand's swizzle, to the destination.
func (e *emitter) writeResult(in *Instruction, call string, scalar bool, b *Binding, sat bool) {
	d := &in.Operands[0]
	t := returnValType(b)
	if scalar {
		e.write(d, call, t, sat)
		return
	}
	resOp := e.resourceOperand(in)
	cs := picks(resOp, destMask(d))
	n := 4
	if b != nil {
		n = b.Components()
	}
	for _, c := range cs {
		if int(c) >= n {
			// the view returns fewer components: pad as the hardware does
			call = fmt.Sprintf("%s(%s, %s)", vecType(t, 4), call, padding(n, t))
			n = 4
			break
		}
	}
	if n == 1 {
		e.write(d, call, t, sat)
		return
	}
	e.write(d, call+"."+letters(cs), t, sat)
}

// padding is what fills a narrower view's missing components: 0 and 1 for
// alpha.
func padding(n int, t valType) string {
	one := "1"
	if t == tFloat {
		one = "1.0"
	}
	parts := []string{"0", "0", "0", one}[n:]
	return strings.Join(parts, ", ")
}

// resourceOperand is the t# or u# operand of a resource instruction.
func (e *emitter) resourceOperand(in *Instruction) *Operand {
	for i := 1; i < len(in.Operands); i++ {
		switch in.Operands[i].Type {
		case OperandResource, OperandUAV:
			return &in.Operands[i]
		}
	}
	return &in.Operands[len(in.Operands)-1]
}

// returnValType is the type a typed view's reads return.
func returnValType(b *Binding) valType {
	if b == nil {
		return tFloat
	}
	switch b.Return {
	case ReturnSint:
		return tInt
	case ReturnUint:
		return tUint
	}
	return tFloat
}

// gatherChannels name Gather's component by the sampler operand's selection.
var gatherChannels = [...]string{"Red", "Green", "Blue", "Alpha"}

// gatherStatement writes gather4, gather4_c, gather4_po and gather4_po_c.
func (e *emitter) gatherStatement(in *Instruction, sat bool) {
	po := in.Op == OpGather4Po || in.Op == OpGather4PoC
	resAt := 2
	if po {
		resAt = 3
	}
	res, b := e.resourceName(&in.Operands[resAt])
	smpOp := &in.Operands[resAt+1]
	smp, _ := e.resourceName(smpOp)
	if b == nil || smp == "" {
		return
	}
	n := coords[b.Dim]
	method := "Gather"
	if in.Op == OpGather4C || in.Op == OpGather4PoC {
		method = "GatherCmp"
	}
	method += gatherChannels[smpOp.Swizzle[0]&3]
	args := []string{smp, e.read(&in.Operands[1], first(&in.Operands[1], n[0]), tFloat)}
	if in.Op == OpGather4C || in.Op == OpGather4PoC {
		args = append(args, e.read(&in.Operands[resAt+2], picks(&in.Operands[resAt+2], 1), tFloat))
	}
	off := offsetArg(in, n[1])
	if po {
		off = ", " + e.read(&in.Operands[2], first(&in.Operands[2], n[1]), tInt)
	}
	call := fmt.Sprintf("%s.%s(%s%s)", res, method, strings.Join(args, ", "), off)
	b4 := *b
	b4.Flags |= bindComponentCountBits // a gather returns four values
	e.writeResult(in, call, false, &b4, sat)
}

// ldStatement writes ld, ld_ms and ld_uav_typed: a typed load by integer
// address.
func (e *emitter) ldStatement(in *Instruction, sat bool) {
	resOp := e.resourceOperand(in)
	res, b := e.resourceName(resOp)
	if b == nil {
		return
	}
	n := coords[b.Dim]
	addr := n[0]
	switch {
	case in.Op == OpLd && b.Dim != DimBuffer && b.Dim != DimBufferEx && b.Dim != DimTexture2DMS && b.Dim != DimTexture2DMSArray:
		addr++ // the mip level
	case in.Op == OpLdUavTyped:
		addr = uavCoords(b.Dim)
	}
	var call string
	switch in.Op {
	case OpLdUavTyped:
		call = fmt.Sprintf("%s[%s]", res, e.read(&in.Operands[1], first(&in.Operands[1], addr), tUint))
	case OpLdMs:
		coord := e.read(&in.Operands[1], first(&in.Operands[1], addr), tInt)
		sample := e.read(&in.Operands[3], picks(&in.Operands[3], 1), tInt)
		call = fmt.Sprintf("%s.Load(%s, %s%s)", res, coord, sample, offsetArg(in, n[1]))
	default:
		coord := e.read(&in.Operands[1], first(&in.Operands[1], addr), tInt)
		call = fmt.Sprintf("%s.Load(%s%s)", res, coord, offsetArg(in, n[1]))
	}
	e.writeResult(in, call, false, b, sat)
}

// uavCoords is how many address components a typed UAV of a dimension takes.
func uavCoords(d Dimension) int {
	switch d {
	case DimTexture1DArray, DimTexture2D:
		return 2
	case DimTexture2DArray, DimTexture3D:
		return 3
	}
	return 1
}

// storeTypedStatement writes store_uav_typed: u#[address] = value.
func (e *emitter) storeTypedStatement(in *Instruction) {
	res, b := e.resourceName(&in.Operands[0])
	if b == nil {
		return
	}
	n := b.Components()
	addr := e.read(&in.Operands[1], first(&in.Operands[1], uavCoords(b.Dim)), tUint)
	val := e.read(&in.Operands[2], first(&in.Operands[2], n), returnValType(b))
	e.line("%s[%s] = %s;", res, addr, val)
}

// resinfoOuts are the GetDimensions out arguments per dimension, as the
// components of resinfo's result they fill.
var resinfoOuts = map[Dimension]string{
	DimTexture1D: "xw", DimTexture1DArray: "xyw", DimTexture2D: "xyw", DimTexture2DArray: "xyzw",
	DimTexture3D: "xyzw", DimTextureCube: "xyw", DimTextureCubeArray: "xyzw", DimTexture2DMS: "xyw",
	DimTexture2DMSArray: "xyzw",
}

// resinfoStatement writes resinfo: dest, mip, resource; the result is width,
// height, depth or array size, mip count, as floats (or their reciprocals but
// the count) or uints.
func (e *emitter) resinfoStatement(in *Instruction, sat bool) {
	res, b := e.resourceName(&in.Operands[2])
	if b == nil {
		return
	}
	outs, ok := resinfoOuts[b.Dim]
	if !ok {
		e.unknown[fmt.Sprintf("resinfo on dimension %d", b.Dim)] = true
		return
	}
	mode := in.ResinfoReturn()
	t := tFloat
	if mode == ResinfoUint {
		t = tUint
	}
	// a UAV or a multisampled texture has one mip level: no level to ask
	// for, and none counted (a multisampled one counts its samples there)
	var args []string
	uav := b.Input.IsUAV()
	ms := b.Dim == DimTexture2DMS || b.Dim == DimTexture2DMSArray
	if !uav && !ms {
		args = append(args, e.read(&in.Operands[1], picks(&in.Operands[1], 1), tUint))
	}
	for _, c := range outs {
		if !uav || c != 'w' {
			args = append(args, "dxbc_dims."+string(c))
		}
	}
	e.line("{")
	e.depth++
	e.line("%s dxbc_dims = 0;", vecType(t, 4))
	e.line("%s.GetDimensions(%s);", res, strings.Join(args, ", "))
	if uav || ms {
		e.line("dxbc_dims.w = %s;", literal(1, t))
	}
	if mode == ResinfoRcpFloat {
		e.line("dxbc_dims.xyz = rcp(dxbc_dims.xyz);")
	}
	e.write(&in.Operands[0], "dxbc_dims."+letters(picks(&in.Operands[2], destMask(&in.Operands[0]))), t, sat)
	e.depth--
	e.line("}")
}

// bufinfoStatement writes bufinfo: a buffer's element count, or a raw
// buffer's byte count.
func (e *emitter) bufinfoStatement(in *Instruction) {
	res, b := e.resourceName(&in.Operands[1])
	if b == nil {
		return
	}
	e.line("{")
	e.depth++
	e.line("uint4 dxbc_dims = 0;")
	switch b.Input {
	case InputStructured, InputUAVStructured, InputUAVAppendStructured, InputUAVConsumeStructured,
		InputUAVStructuredWithCounter:
		e.line("%s.GetDimensions(dxbc_dims.x, dxbc_dims.y);", res)
		e.line("dxbc_dims.y = 0;")
	default:
		e.line("%s.GetDimensions(dxbc_dims.x);", res)
	}
	e.write(&in.Operands[0], "dxbc_dims."+letters(picks(&in.Operands[1], destMask(&in.Operands[0]))), tUint, false)
	e.depth--
	e.line("}")
}

// lodStatement writes lod: the clamped and the unclamped level of detail.
func (e *emitter) lodStatement(in *Instruction, sat bool) {
	res, b := e.resourceName(&in.Operands[2])
	smp, _ := e.resourceName(&in.Operands[3])
	if b == nil || smp == "" {
		return
	}
	coord := e.read(&in.Operands[1], first(&in.Operands[1], coords[b.Dim][0]), tFloat)
	call := fmt.Sprintf("float4(%s.CalculateLevelOfDetail(%s, %s), %s.CalculateLevelOfDetailUnclamped(%s, %s), 0, 0)",
		res, smp, coord, res, smp, coord)
	e.write(&in.Operands[0], call+"."+letters(picks(&in.Operands[2], destMask(&in.Operands[0]))), tFloat, sat)
}

// sampleinfoStatement writes sampleinfo: a multisampled texture's sample
// count, in x.
func (e *emitter) sampleinfoStatement(in *Instruction, sat bool) {
	t := tFloat
	if in.SampleinfoUint() {
		t = tUint
	}
	if in.Operands[1].Type == operandRasterizer {
		call := convert("uint4(GetRenderTargetSampleCount(), 0, 0, 0)", tUint, t)
		if t == tFloat {
			call = "float4(GetRenderTargetSampleCount(), 0, 0, 0)"
		}
		e.write(&in.Operands[0], call+"."+letters(picks(&in.Operands[1], destMask(&in.Operands[0]))), t, sat)
		return
	}
	res, b := e.resourceName(&in.Operands[1])
	if b == nil {
		return
	}
	args := "dxbc_dims.y, dxbc_dims.z, dxbc_dims.x"
	if b.Dim == DimTexture2DMSArray {
		args = "dxbc_dims.y, dxbc_dims.z, dxbc_dims.w, dxbc_dims.x"
	}
	e.line("{")
	e.depth++
	e.line("%s dxbc_dims = 0;", vecType(t, 4))
	e.line("%s.GetDimensions(%s);", res, args)
	e.line("dxbc_dims.yzw = 0;")
	e.write(&in.Operands[0], "dxbc_dims."+letters(picks(&in.Operands[1], destMask(&in.Operands[0]))), t, sat)
	e.depth--
	e.line("}")
}

// sampleposStatement writes samplepos: where a sample of a multisampled
// texture lies in its pixel.
func (e *emitter) sampleposStatement(in *Instruction, sat bool) {
	idx := e.read(&in.Operands[2], picks(&in.Operands[2], 1), tInt)
	call := fmt.Sprintf("float4(GetRenderTargetSamplePosition(%s), 0, 0)", idx)
	if in.Operands[1].Type != operandRasterizer {
		res, b := e.resourceName(&in.Operands[1])
		if b == nil {
			return
		}
		call = fmt.Sprintf("float4(%s.GetSamplePosition(%s), 0, 0)", res, idx)
	}
	e.write(&in.Operands[0], call+"."+letters(picks(&in.Operands[1], destMask(&in.Operands[0]))), tFloat, sat)
}

// operandRasterizer is the render target's own sample pattern, as sampleinfo
// and samplepos read it.
const operandRasterizer OperandType = 14

// evalFunctions are the intrinsics that evaluate an attribute elsewhere in
// the pixel than its center: eval_centroid, eval_sample_index (a sample
// index) and eval_snapped (an offset in sixteenths of a pixel).
var evalFunctions = map[Opcode]string{
	OpEvalCentroid: "EvaluateAttributeCentroid", OpEvalSampleIndex: "EvaluateAttributeAtSample",
	OpEvalSnapped: "EvaluateAttributeSnapped",
}

// evalStatement writes eval_*: dest, input[, sample index or offset].
func (e *emitter) evalStatement(in *Instruction, sat bool) {
	d := &in.Operands[0]
	args := []string{e.read(&in.Operands[1], picks(&in.Operands[1], destMask(d)), tFloat)}
	switch in.Op {
	case OpEvalSampleIndex:
		args = append(args, e.read(&in.Operands[2], picks(&in.Operands[2], 1), tUint))
	case OpEvalSnapped:
		args = append(args, e.read(&in.Operands[2], first(&in.Operands[2], 2), tInt))
	}
	e.write(d, fmt.Sprintf("%s(%s)", evalFunctions[in.Op], strings.Join(args, ", ")), tFloat, sat)
}
