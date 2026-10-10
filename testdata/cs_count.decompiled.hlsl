// Decompiled by github.com/McReaper/t7_dxbc

cbuffer Debug : register(b0)
{
  float4 packed : packoffset(c0);
}

StructuredBuffer<float3> points : register(t0);
RWStructuredBuffer<float> nearest : register(u0);

[numthreads(4, 1, 1)]
void cs_main(uint3 vThreadID : SV_DispatchThreadID)
{
  float4 r0, r1;
  uint4 i1;
  
  r0.xyz = float3(vThreadID.xyz);
  r0.w = 1e+30;
  i1.x = asuint(0);
  while (true) {
    i1.y = i1.x >= asuint(packed.x) ? 0xffffffffu : 0u;
    if (i1.y != 0) break;
    {
      float3 dxbc_v = points[i1.x].xyz;
      r1.zw = dxbc_v.yz;
      i1.y = asuint(dxbc_v.x);
    }
    {
      float3 dxbc_v = (-r0.xyz) + (float3(asfloat(i1.y), r1.z, r1.w));
      r1.zw = dxbc_v.yz;
      i1.y = asuint(dxbc_v.x);
    }
    i1.y = asuint(dot(float3(asfloat(i1.y), r1.z, r1.w), float3(asfloat(i1.y), r1.z, r1.w)));
    i1.y = asuint(sqrt(asfloat(i1.y)));
    r0.w = min(r0.w, asfloat(i1.y));
    i1.x = asuint(asint(i1.x) + 1);
  }
  nearest[vThreadID.x] = r0.w;
  return;
}
