// Decompiled by github.com/McReaper/dxbc

globallycoherent RWByteAddressBuffer coherent : register(u0);
RWStructuredBuffer<uint> local : register(u1);
groupshared uint g0[1];
groupshared uint g1[32][1];

uint dxbc_bfi(uint w, uint o, uint i, uint b) { w &= 31; o &= 31; uint m = ((1u << w) - 1) << o; return ((i << o) & m) | (b & ~m); }

[numthreads(32, 1, 1)]
void main(uint3 vThreadGroupID : SV_GroupID,
  uint vThreadIDInGroupFlattened : SV_GroupIndex)
{
  float4 r1;
  uint4 i0, i1;
  
  if (vThreadIDInGroupFlattened == 0) {
    g0[(0u >> 2) + 0] = 0u;
  }
  GroupMemoryBarrierWithGroupSync();
  i0.xy = asuint(asint(vThreadIDInGroupFlattened.xx) + (int2(1, 3)));
  {
    uint dxbc_before;
    InterlockedAdd(g0[0u >> 2], i0.x, dxbc_before);
    i1.x = dxbc_before;
  }
  g1[vThreadIDInGroupFlattened][0] = i1.x;
  GroupMemoryBarrier();
  i0.z = asuint(asint(vThreadGroupID.x) * 32 + asint(vThreadIDInGroupFlattened));
  i0.x = i0.x & 31u;
  i0.x = g1[i0.x][0];
  local[i0.z] = i0.x;
  DeviceMemoryBarrier();
  i0.x = asuint(asint(vThreadIDInGroupFlattened) << 2);
  i0.w = vThreadIDInGroupFlattened ^ 1u;
  i0.w = asuint(asint(vThreadGroupID.x) * 32 + asint(i0.w));
  i0.w = local[i0.w];
  coherent.Store(i0.x, i0.w);
  DeviceMemoryBarrierWithGroupSync();
  i0.x = dxbc_bfi(5u, 2u, i0.y, 0u);
  i0.x = coherent.Load(i0.x);
  AllMemoryBarrier();
  i0.y = g0[(0u >> 2) + 0];
  i0.x = asuint(asint(i0.y) + asint(i0.x));
  g1[vThreadIDInGroupFlattened][0] = i0.x;
  AllMemoryBarrierWithGroupSync();
  i0.x = asuint((-asint(vThreadIDInGroupFlattened)) + 31);
  i0.x = g1[i0.x][0];
  local[i0.z] = i0.x;
  return;
}
