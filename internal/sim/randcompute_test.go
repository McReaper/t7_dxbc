//go:build windows

package sim

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// randomComputeShader writes a random compute shader: the expressions of
// randomShader, reading structured and raw buffers (an array inside the
// element, at a computed index) and group-shared memory, and statements that
// write group memory, wait at barriers and change memory atomically, then
// write their results to every kind of UAV.
func randomComputeShader(seed uint64) string {
	g := &shaderGen{r: rand.New(rand.NewPCG(seed, seed^0xD1B54A32D192ED03)), compute: true}
	var body strings.Builder
	g.out = &body
	g.locals = map[string]int{}
	for range 3 + g.r.IntN(5) {
		g.statement(1)
	}
	return fmt.Sprintf(`struct Item
{
	float3 pos;
	uint flags;
	float vals[4];
	uint2 ids;
};

cbuffer Params : register(b0)
{
	float4 fv[4];
	uint4 uv[2];
	int4 iv[2];
	float fs;
	uint us;
};
StructuredBuffer<Item> src : register(t0);
ByteAddressBuffer table : register(t1);
RWStructuredBuffer<Item> items : register(u0);
RWByteAddressBuffer raw : register(u1);
RWTexture2D<float4> img : register(u2);
RWTexture2D<uint> counts : register(u3);
groupshared uint gs[32];
groupshared float4 gv[32];

[numthreads(8, 4, 1)]
void main(uint3 tid : SV_DispatchThreadID, uint3 gt : SV_GroupThreadID, uint gi : SV_GroupIndex)
{
	float4 inF = float4(tid.xyz, gi) * 0.37 - 1;
	uint4 inU = uint4(tid.xy, gt.x, gi);
	int4 inI = int4(gt.xyz, gi) - 5;
	float4 lf = inF;
	uint4 lu = inU;
	int4 li = inI;
	gs[gi] = lu.x * 3 + lu.y;
	gv[gi] = lf;
	GroupMemoryBarrierWithGroupSync();
%s	items[gi].pos = lf.xyz;
	items[gi].vals[lu.y & 3] = lf.w;
	items[gi].ids = lu.zw ^ asuint(li.xy);
	raw.Store4(gi * 16, lu ^ asuint(li));
	img[tid.xy] = lf;
	counts[gt.xy] = lu.x;
}
`, body.String())
}

// memoryLeaf reads a buffer or group memory.
func (g *shaderGen) memoryLeaf(kind, n int) string {
	idx := fmt.Sprintf("(lu.%s & 15)", g.swizzle(1))
	var v string
	switch g.r.IntN(5) {
	case 0:
		v = fmt.Sprintf("src[%s].vals[lu.%s & 3]", idx, g.swizzle(1)) // a computed offset into the element
	case 1:
		v = fmt.Sprintf("src[%s].pos.%s", idx, g.pick("x", "y", "z"))
	case 2:
		v = fmt.Sprintf("src[%s].ids.%s", idx, g.pick("x", "y"))
	case 3:
		v = fmt.Sprintf("gs[(gi + %d) & 31]", 1+g.r.IntN(31)) // another thread's value
	default:
		return fmt.Sprintf("((%s)gv[(gi + %d) & 31].%s)", vec(kind, n), 1+g.r.IntN(31), g.swizzle(n))
	}
	if g.r.IntN(4) == 0 {
		v = fmt.Sprintf("asfloat(table.Load(%s * 4))", idx)
	}
	return fmt.Sprintf("((%s)%s)", vec(kind, n), v)
}

// memoryStatement writes group memory, waits at a barrier (only at the top
// level, where every thread reaches it), or changes memory atomically.
func (g *shaderGen) memoryStatement(ind string, depth int) {
	switch k := g.r.IntN(6); {
	case k == 0:
		fmt.Fprintf(g.out, "%sgs[(gi + %d) & 31] = %s;\n", ind, g.r.IntN(32), g.expr(kUint, 1, 2))
	case k == 1:
		fmt.Fprintf(g.out, "%sgv[gi].%s = %s;\n", ind, "xyzw"[g.r.IntN(4):][:1], g.expr(kFloat, 1, 2))
	case k == 2 && depth == 1:
		fmt.Fprintf(g.out, "%s%s();\n", ind, g.pick("GroupMemoryBarrierWithGroupSync", "AllMemoryBarrierWithGroupSync",
			"DeviceMemoryBarrierWithGroupSync"))
	case k == 3:
		fmt.Fprintf(g.out, "%s%s(gs[lu.%s & 31], %s, lu.%s);\n", ind, g.pick("InterlockedAdd", "InterlockedMax", "InterlockedOr",
			"InterlockedXor", "InterlockedMin", "InterlockedAnd", "InterlockedExchange"), g.swizzle(1), g.expr(kUint, 1, 2), g.swizzle(1))
	case k == 4:
		fmt.Fprintf(g.out, "%sInterlocked%s(items[lu.%s & 7].flags, %s);\n", ind, g.pick("Add", "Max", "Or", "Min"),
			g.swizzle(1), g.expr(kUint, 1, 2))
	default:
		fmt.Fprintf(g.out, "%sraw.Interlocked%s((lu.%s & 31) * 4, %s, lu.%s);\n", ind, g.pick("Add", "Max", "Xor", "And"),
			g.swizzle(1), g.expr(kUint, 1, 2), g.swizzle(1))
	}
}
