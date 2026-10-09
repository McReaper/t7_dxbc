// Decompiled by github.com/McReaper/dxbc

struct Pair
{
  float a;
  uint b;
};

struct Bin
{
  uint count;
  float weight;
  uint slots[4];
};

cbuffer Params : register(b0)
{
  uint binCount : packoffset(c0);
  uint pick : packoffset(c0.y);
  float scale : packoffset(c0.z);
  int bias : packoffset(c0.w);
}

StructuredBuffer<Pair> source : register(t0);
ByteAddressBuffer table : register(t1);
Buffer<float4> typed : register(t2);
RWByteAddressBuffer raw : register(u0);
RWStructuredBuffer<Bin> bins : register(u1);
RWTexture2D<uint> counts : register(u2);
RWStructuredBuffer<Pair> pairs : register(u3);
RWTexture2DArray<float4> layers : register(u4);
groupshared uint g0[64][1];
groupshared uint g1[64][2];

uint dxbc_bfi(uint w, uint o, uint i, uint b) { w &= 31; o &= 31; uint m = ((1u << w) - 1) << o; return ((i << o) & m) | (b & ~m); }
uint dxbc_InterlockedAdd_bins(uint dxbc_i, uint dxbc_o0, uint dxbc_a)
{
  uint dxbc_old = 0;
  if (dxbc_o0 == 0u) { InterlockedAdd(bins[dxbc_i].count, dxbc_a, dxbc_old); return dxbc_old; }
  if (dxbc_o0 == 4u) { return 0; }
  if (dxbc_o0 >= 8u && dxbc_o0 < 24u) {
    uint dxbc_o1 = dxbc_o0 - 8u;
    if (dxbc_o1 < 16u) {
      uint dxbc_k2 = dxbc_o1 / 4u;
      uint dxbc_o2 = dxbc_o1 % 4u;
      if (dxbc_o2 == 0u) { InterlockedAdd(bins[dxbc_i].slots[dxbc_k2], dxbc_a, dxbc_old); return dxbc_old; }
    }
  }
  return dxbc_old;
}
void dxbc_store_bins(uint dxbc_i, uint dxbc_o0, uint dxbc_value)
{
  if (dxbc_o0 == 0u) { bins[dxbc_i].count = dxbc_value; return; }
  if (dxbc_o0 == 4u) { bins[dxbc_i].weight = asfloat(dxbc_value); return; }
  if (dxbc_o0 >= 8u && dxbc_o0 < 24u) {
    uint dxbc_o1 = dxbc_o0 - 8u;
    if (dxbc_o1 < 16u) {
      uint dxbc_k2 = dxbc_o1 / 4u;
      uint dxbc_o2 = dxbc_o1 % 4u;
      if (dxbc_o2 == 0u) { bins[dxbc_i].slots[dxbc_k2] = dxbc_value; return; }
    }
  }
}

[numthreads(8, 4, 1)]
void main(uint3 vThreadID : SV_DispatchThreadID,
  uint vThreadIDInGroupFlattened : SV_GroupIndex)
{
  uint4 i0, i1, i2, i3;
  
  i0.x = asuint(asint(vThreadID.x) * 3 + asint(vThreadID.y));
  g0[vThreadIDInGroupFlattened][0] = i0.x;
  i0.xy = uint2(asuint(source[vThreadIDInGroupFlattened].a), source[vThreadIDInGroupFlattened].b);
  i1.x = asuint(asfloat(i0.x) * scale);
  i1.y = asuint(asint(i0.y) + asint(pick));
  g1[vThreadIDInGroupFlattened][0] = i1.x;
  g1[vThreadIDInGroupFlattened][1] = i1.y;
  GroupMemoryBarrierWithGroupSync();
  i0.xy = asuint(asint(vThreadIDInGroupFlattened.xx) + (int2(1, 3)));
  i0.xy = i0.xy & (uint2(31u, 31u));
  i0.x = g0[i0.x][0];
  i0.w = asuint(asint(vThreadIDInGroupFlattened) + asint(pick));
  i1.x = i0.w & 31u;
  i0.w = dxbc_bfi(2u, 2u, i0.w, 0u);
  i2.w = asuint(asint(i0.w) + 8);
  i1.xy = uint2(g1[i1.x][0], g1[i1.x][1]);
  i2.z = asuint(0);
  i2.xy = vThreadIDInGroupFlattened.xx & (uint2(7u, 3u));
  InterlockedAdd(g0[i2.x][i2.z >> 2], i0.x);
  i0.z = asuint(0);
  {
    uint dxbc_before;
    InterlockedMax(g0[i0.y][i0.z >> 2], i1.y, dxbc_before);
    i3.x = dxbc_before;
  }
  GroupMemoryBarrierWithGroupSync();
  i0.y = asuint(float(i3.x));
  i0.y = asuint(asfloat(i0.y) + asfloat(i1.x));
  i0.z = g0[vThreadIDInGroupFlattened][0];
  dxbc_InterlockedAdd_bins(i2.y, i2.w, 1u);
  i0.w = asuint(1 << asint(vThreadIDInGroupFlattened));
  InterlockedOr(bins[0u].count, i0.w);
  {
    uint dxbc_before;
    InterlockedExchange(bins[1u].count, i0.x, dxbc_before);
    i3.x = dxbc_before;
  }
  {
    uint dxbc_before;
    InterlockedCompareExchange(bins[2u].slots[1], 0u, vThreadIDInGroupFlattened, dxbc_before);
    i3.x = dxbc_before;
  }
  InterlockedCompareStore(bins[3u].slots[2], 7u, vThreadIDInGroupFlattened);
  i0.w = i0.x ^ i3.x;
  bins[i2.y].weight = asfloat(i0.y);
  dxbc_store_bins(i2.y, i2.w, i0.w);
  i1.xzw = asuint(asint(vThreadIDInGroupFlattened.xxx) << (int3(2, 4, 3)));
  raw.InterlockedAdd(i1.x, i0.x);
  i0.x = i0.x | 240u;
  InterlockedAnd(counts[vThreadID.xy], i0.x);
  i0.xyw = asuint(asint(i1.xwx) + (int3(128, 256, 1024)));
  i2.xyzw = table.Load4(i1.z).xyzw;
  raw.InterlockedXor(i0.x, i1.y);
  i0.xy = raw.Load2(i0.y).xy;
  i1.xyz = asuint(asint(i0.xxy) + asint(i2.xyz));
  i0.x = asuint(asint(vThreadIDInGroupFlattened) * 12 + 512);
  raw.Store3(i0.x, i1.xyz);
  i0.x = asuint(asint(i0.z) + bias);
  i1.y = asuint(asint(i0.z) + 8);
  raw.Store(i0.w, i0.x);
  i0.x = counts[vThreadID.xy];
  i0.x = asuint(asint(i2.w) + asint(i0.x));
  i2.xyzw = asuint(asint(vThreadID.xyyy) + (int4(16, 0, 0, 0)));
  counts[i2.xy] = i0.x;
  i0.xy = vThreadID.xy;
  i0.zw = asuint(float2(0, 0));
  i0.xyzw = asuint(layers[i0.xyz].xyzw);
  {
    uint4 dxbc_dims = 0;
    typed.GetDimensions(dxbc_dims.x);
    i1.z = dxbc_dims.x;
  }
  i1.z = vThreadIDInGroupFlattened % i1.z;
  i2.xyzw = asuint(typed.Load(asint(i1.z)).xyzw);
  {
    uint4 dxbc_dims = 0;
    pairs.GetDimensions(dxbc_dims.x, dxbc_dims.y);
    dxbc_dims.y = 0;
    i1.z = dxbc_dims.x;
  }
  i1.w = asuint(float(i1.z));
  i0.xyzw = asuint(mad(asfloat(i2.xyzw), asfloat(i1.wwww), asfloat(i0.xyzw)));
  {
    uint4 dxbc_dims = 0;
    layers.GetDimensions(dxbc_dims.x, dxbc_dims.y, dxbc_dims.z);
    dxbc_dims.w = 1u;
    i2.yzw = dxbc_dims.xyz;
  }
  i3.xy = vThreadID.xy % i2.yz;
  i3.zw = vThreadIDInGroupFlattened.xx % i2.ww;
  layers[i3.xyz] = asfloat(i0.xyzw);
  i0.x = pairs.IncrementCounter();
  i0.x = i0.x % i1.z;
  i0.y = asuint(float(binCount));
  i1.x = asuint(asfloat(i0.y) + asfloat(i2.x));
  pairs[i0.x].a = asfloat(i1.x);
  pairs[i0.x].b = i1.y;
  return;
}
