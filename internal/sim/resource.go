package sim

import (
	"fmt"
	"math"

	"github.com/McReaper/dxbc"
)

// The simulated resources' sizes.
const (
	texWidth, texHeight, texDepth, texElements, texMips = 256, 128, 32, 6, 9
	bufElements                                         = 64
)

// mix hashes words with the world's seed.
func (w *World) mix(xs ...uint32) uint32 {
	h := w.Seed*0x9E3779B9 + 0x7F4A7C15
	for _, x := range xs {
		h ^= x
		h *= 0x85EBCA6B
		h ^= h >> 13
		h *= 0xC2B2AE35
		h ^= h >> 16
	}
	return h
}

// hashed is a simulated memory value: a float in [-1, 1] or an integer in
// [0, 16).
func (w *World) hashed(isFloat bool, xs ...uint32) uint32 {
	h := w.mix(xs...)
	if isFloat {
		return math.Float32bits(float32(h%2001)/1000 - 1)
	}
	return h % 16
}

// texel is a texture's value: smooth in its coordinates and level of detail,
// so that sampling at coordinates computed in another order changes it
// little.
func (w *World) texel(reg uint32, comp int, c [4]float32, lod float32) float32 {
	k := float64(comp + 1)
	phase := float64(w.Seed%97)*0.1 + float64(reg)*0.37
	a := 2.7*float64(c[0])*k + 1.9*float64(c[1]) + 1.3*float64(c[2])*k + 0.7*float64(c[3]) + 0.21*float64(lod)
	return float32(0.5 + 0.45*math.Sin(a+phase))
}

// binding is the declaration of a resource operand.
func (t *thread) binding(op *dxbc.Operand) (*dxbc.Binding, error) {
	b := t.m.binds[view{op.Type, uint32(t.index(op, 0))}]
	if b == nil {
		return nil, fmt.Errorf("no binding for operand type %d register %d", op.Type, t.index(op, 0))
	}
	return b, nil
}

// coordCounts are how many address components each view dimension reads,
// and how many offset components it takes.
var coordCounts = map[dxbc.Dimension][2]int{
	dxbc.DimBuffer: {1, 0}, dxbc.DimTexture1D: {1, 1}, dxbc.DimTexture1DArray: {2, 1}, dxbc.DimTexture2D: {2, 2},
	dxbc.DimTexture2DArray: {3, 2}, dxbc.DimTexture2DMS: {2, 2}, dxbc.DimTexture2DMSArray: {3, 2},
	dxbc.DimTexture3D: {3, 3}, dxbc.DimTextureCube: {3, 0}, dxbc.DimTextureCubeArray: {4, 0}, dxbc.DimBufferEx: {1, 0},
}

// coords reads a float address: the components the view reads, moved by the
// offsets, the others 0.
func (t *thread) coords(op *dxbc.Operand, b *dxbc.Binding, off [3]int32) [4]float32 {
	f := t.fsrc(op)
	n := coordCounts[b.Dim]
	var c [4]float32
	copy(c[:n[0]], f[:n[0]])
	sizes := [3]float32{texWidth, texHeight, texDepth}
	for i := range n[1] {
		c[i] += float32(off[i]) / sizes[i]
	}
	return c
}

// sampleLod is the level of detail a sample instruction asks for.
func (t *thread) sampleLod(in *dxbc.Instruction) float32 {
	switch in.Op {
	case dxbc.OpSampleB, dxbc.OpSampleL:
		return t.fsrc(&in.Operands[4])[0]
	case dxbc.OpSampleD:
		dx, dy := t.fsrc(&in.Operands[4]), t.fsrc(&in.Operands[5])
		var l float32
		for i := range 3 {
			l += dx[i]*dx[i] + 2*dy[i]*dy[i]
		}
		return l
	}
	return 0
}

// compare is a comparison sample's result: smooth, 1 well above the
// reference.
func compare(texel, ref float32) float32 {
	return math.Float32frombits(saturate(math.Float32bits((texel-ref)*4 + 0.5)))
}

func sample(t *thread, in *dxbc.Instruction) error {
	b, err := t.binding(&in.Operands[2])
	if err != nil {
		return err
	}
	c := t.coords(&in.Operands[1], b, in.Offsets())
	lod := t.sampleLod(in)
	reg := uint32(t.index(&in.Operands[2], 0))
	var f [4]float32
	for i := range f {
		f[i] = t.w.texel(reg, i, c, lod)
	}
	if in.Op == dxbc.OpSampleC || in.Op == dxbc.OpSampleCLz {
		r := compare(f[0], t.fsrc(&in.Operands[4])[0])
		f = [4]float32{r, r, r, r}
	}
	return t.storeFetched(in, &in.Operands[2], floats(f), sat(in))
}

func floats(f [4]float32) Value {
	var v Value
	for i := range v {
		v[i] = math.Float32bits(f[i])
	}
	return v
}

// storeFetched writes what a resource returned, through the resource
// operand's swizzle.
func (t *thread) storeFetched(in *dxbc.Instruction, res *dxbc.Operand, v Value, saturated bool) error {
	sw := v
	if res.Selection == dxbc.SelectSwizzle || res.Selection == dxbc.SelectOne {
		for i := range sw {
			s := res.Swizzle[i]
			if res.Selection == dxbc.SelectOne {
				s = res.Swizzle[0]
			}
			sw[i] = v[s&3]
		}
	}
	return t.store(&in.Operands[0], sw, saturated)
}

func gather(t *thread, in *dxbc.Instruction) error {
	po := in.Op == dxbc.OpGather4Po || in.Op == dxbc.OpGather4PoC
	resAt := 2
	if po {
		resAt = 3
	}
	res, smp := &in.Operands[resAt], &in.Operands[resAt+1]
	b, err := t.binding(res)
	if err != nil {
		return err
	}
	off := in.Offsets()
	if po {
		o := t.src(&in.Operands[2])
		off = [3]int32{int32(o[0]), int32(o[1]), int32(o[2])}
	}
	c := t.coords(&in.Operands[1], b, off)
	channel := int(smp.Swizzle[0] & 3)
	reg := uint32(t.index(res, 0))
	corners := [4][2]float32{{-0.5, 0.5}, {0.5, 0.5}, {0.5, -0.5}, {-0.5, -0.5}}
	var f [4]float32
	for i, d := range corners {
		cc := c
		cc[0] += d[0] / texWidth
		cc[1] += d[1] / texHeight
		f[i] = t.w.texel(reg, channel, cc, 0)
		if in.Op == dxbc.OpGather4C || in.Op == dxbc.OpGather4PoC {
			f[i] = compare(f[i], t.fsrc(&in.Operands[resAt+2])[0])
		}
	}
	return t.storeFetched(in, res, floats(f), sat(in))
}

// isFloatView reports whether a typed view holds floats.
func isFloatView(b *dxbc.Binding) bool {
	return b.Return != dxbc.ReturnSint && b.Return != dxbc.ReturnUint
}

// fetchTyped is a typed view's texel at integer coordinates: written UAV
// memory, else simulated. A view of fewer components returns 0 and, for
// alpha, 1 in the others.
func (t *thread) fetchTyped(op *dxbc.Operand, b *dxbc.Binding, key [4]uint32) Value {
	reg := uint32(t.index(op, 0))
	var v Value
	for i := range v {
		if i >= b.Components() {
			if i == 3 {
				v[i] = 1
				if isFloatView(b) {
					v[i] = math.Float32bits(1)
				}
			}
			continue
		}
		if op.Type == dxbc.OperandUAV {
			if x, ok := t.w.UAV[Cell{reg, key[0], key[1], key[2], uint32(i)}]; ok {
				v[i] = x
				continue
			}
		}
		v[i] = t.w.hashed(isFloatView(b), uint32(op.Type), reg, key[0], key[1], key[2], key[3], uint32(i))
	}
	return v
}

func ld(t *thread, in *dxbc.Instruction) error {
	res := &in.Operands[2]
	b, err := t.binding(res)
	if err != nil {
		return err
	}
	a := t.src(&in.Operands[1])
	n := coordCounts[b.Dim]
	var key [4]uint32
	copy(key[:], a[:n[0]])
	off := in.Offsets()
	for i := range n[1] {
		key[i] += uint32(off[i])
	}
	switch {
	case in.Op == dxbc.OpLdMs:
		key[3] = t.src(&in.Operands[3])[0]
	case in.Op == dxbc.OpLd && b.Dim != dxbc.DimBuffer && b.Dim != dxbc.DimBufferEx:
		key[3] = a[n[0]] // the mip level
	}
	return t.storeFetched(in, res, t.fetchTyped(res, b, key), false)
}

func ldUAVTyped(t *thread, in *dxbc.Instruction) error {
	res := &in.Operands[2]
	b, err := t.binding(res)
	if err != nil {
		return err
	}
	a := t.src(&in.Operands[1])
	var key [4]uint32
	copy(key[:], a[:uavCoordCount(b.Dim)])
	return t.storeFetched(in, res, t.fetchTyped(res, b, key), false)
}

func uavCoordCount(d dxbc.Dimension) int {
	switch d {
	case dxbc.DimTexture1DArray, dxbc.DimTexture2D:
		return 2
	case dxbc.DimTexture2DArray, dxbc.DimTexture3D:
		return 3
	}
	return 1
}

func storeUAVTyped(t *thread, in *dxbc.Instruction) error {
	dst := &in.Operands[0]
	b, err := t.binding(dst)
	if err != nil {
		return err
	}
	a, v := t.src(&in.Operands[1]), t.src(&in.Operands[2])
	var key [3]uint32
	copy(key[:], a[:uavCoordCount(b.Dim)])
	reg := uint32(t.index(dst, 0))
	for i := range b.Components() {
		t.w.UAV[Cell{reg, key[0], key[1], key[2], uint32(i)}] = v[i]
	}
	return nil
}

// dims is what resinfo returns for a view at a mip level: width, height,
// depth or array size, mip count.
func dims(d dxbc.Dimension, mip uint32) [4]uint32 {
	if mip >= texMips {
		return [4]uint32{0, 0, 0, texMips}
	}
	w, h, z := max(uint32(texWidth)>>mip, 1), max(uint32(texHeight)>>mip, 1), max(uint32(texDepth)>>mip, 1)
	switch d {
	case dxbc.DimTexture1D:
		return [4]uint32{w, 0, 0, texMips}
	case dxbc.DimTexture1DArray:
		return [4]uint32{w, texElements, 0, texMips}
	case dxbc.DimTexture2DArray, dxbc.DimTextureCubeArray:
		return [4]uint32{w, h, texElements, texMips}
	case dxbc.DimTexture3D:
		return [4]uint32{w, h, z, texMips}
	case dxbc.DimTexture2DMS:
		return [4]uint32{w, h, 0, 1}
	case dxbc.DimTexture2DMSArray:
		return [4]uint32{w, h, texElements, 1}
	}
	return [4]uint32{w, h, 0, texMips}
}

func resinfo(t *thread, in *dxbc.Instruction) error {
	res := &in.Operands[2]
	b, err := t.binding(res)
	if err != nil {
		return err
	}
	d := dims(b.Dim, t.src(&in.Operands[1])[0])
	if b.Input.IsUAV() {
		d[3] = 1 // one mip level
	}
	var v Value
	for i, x := range d {
		switch in.ResinfoReturn() {
		case dxbc.ResinfoUint:
			v[i] = x
		case dxbc.ResinfoRcpFloat: // the sizes' reciprocals, the mip count as is
			v[i] = math.Float32bits(float32(x))
			if i < 3 {
				v[i] = math.Float32bits(1 / float32(x))
			}
		default:
			v[i] = math.Float32bits(float32(x))
		}
	}
	return t.storeFetched(in, res, v, sat(in))
}

func bufinfo(t *thread, in *dxbc.Instruction) error {
	b, err := t.binding(&in.Operands[1])
	if err != nil {
		return err
	}
	n := uint32(bufElements)
	if b.Input == dxbc.InputByteAddress || b.Input == dxbc.InputUAVByteAddress {
		n = bufElements * 16
	}
	return t.storeFetched(in, &in.Operands[1], Value{n, n, n, n}, false)
}

func lod(t *thread, in *dxbc.Instruction) error {
	return t.storeFetched(in, &in.Operands[2], Value{}, sat(in))
}

// msSamples is a multisampled texture's or render target's sample count,
// and samplePositions the standard pattern of four.
const msSamples = 4

var samplePositions = [msSamples][2]float32{{-0.125, -0.375}, {0.375, -0.125}, {-0.375, 0.125}, {0.125, 0.375}}

func sampleinfo(t *thread, in *dxbc.Instruction) error {
	v := Value{math.Float32bits(msSamples)}
	if in.SampleinfoUint() {
		v = Value{msSamples}
	}
	return t.storeFetched(in, &in.Operands[1], v, sat(in))
}

func samplepos(t *thread, in *dxbc.Instruction) error {
	i := t.src(&in.Operands[2])[0]
	var v Value
	if i < msSamples {
		v = floats([4]float32{samplePositions[i][0], samplePositions[i][1]})
	}
	return t.storeFetched(in, &in.Operands[1], v, sat(in))
}
