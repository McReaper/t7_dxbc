// Decompiled by github.com/McReaper/dxbc

Texture2DMS<float4> ms : register(t0);
RWStructuredBuffer<int> extremes : register(u1);

void main(noperspective float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  float3 in_NORMAL0 : NORMAL0,
  nointerpolation uint in_SV_SampleIndex0 : SV_SampleIndex0,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r1, r2, r3;
  uint4 i0, i1, i2;
  
  i0.xyzw = asuint(int4(in_SV_Position0.xyxy));
  i0.x = asuint(asint(i0.x) + (-900));
  InterlockedMax(extremes[0u], asint(i0.x));
  i1.xyz = asuint(-asint(i0.yzw));
  {
    int dxbc_before;
    InterlockedMin(extremes[1u], asint(i1.x), dxbc_before);
    i2.x = asuint(dxbc_before);
  }
  i0.xy = asuint(max(asint(i0.zw), asint(i1.yz)));
  i0.zw = i0.zw & (uint2(0x80000000u, 0x80000000u));
  i1.x = asuint(float(asint(i2.x)));
  {
    uint4 dxbc_dims = 0;
    ms.GetDimensions(dxbc_dims.x, dxbc_dims.y, dxbc_dims.w);
    dxbc_dims.w = 1u;
    i1.yz = dxbc_dims.xy;
  }
  i1.yz = asuint(max(asint(i1.yz), (-asint(i1.yz))));
  i0.xy = i0.xy % i1.yz;
  i1.yz = asuint(-asint(i0.xy));
  i0.xy = i0.zw ? i1.yz : i0.xy;
  i0.zw = asuint(float2(0, 0));
  {
    uint4 dxbc_dims = 0;
    ms.GetDimensions(dxbc_dims.y, dxbc_dims.z, dxbc_dims.x);
    dxbc_dims.yzw = 0;
    i1.y = dxbc_dims.x;
  }
  i1.z = in_SV_SampleIndex0.x % i1.y;
  i0.xyzw = asuint(ms.Load(asint(i0.xy), asint(i1.z)).xyzw);
  {
    float2 dxbc_v = float4(ms.GetSamplePosition(asint(i1.z)), 0, 0).xy;
    r1.w = dxbc_v.y;
    i1.z = asuint(dxbc_v.x);
  }
  {
    float3 dxbc_v = EvaluateAttributeSnapped(in_NORMAL0.xyz, int2(1, -2));
    r2.yz = dxbc_v.yz;
    i2.x = asuint(dxbc_v.x);
  }
  i1.x = asuint(asfloat(i1.x) + r2.y);
  r3.w = r2.z + asfloat(i1.x);
  i1.x = uint4(GetRenderTargetSampleCount(), 0, 0, 0).x;
  i1.x = in_SV_SampleIndex0.x % i1.x;
  r2.yz = float4(GetRenderTargetSamplePosition(asint(i1.x)), 0, 0).xy;
  i1.xz = asuint((float2(asfloat(i1.z), r1.w)) + r2.yz);
  r2.yz = EvaluateAttributeCentroid(in_TEXCOORD0.xy);
  r3.xy = asfloat(i1.xz) + r2.yz;
  i1.x = asuint(asint(in_SV_SampleIndex0.x) + 1);
  i1.x = i1.x % i1.y;
  i1.x = asuint(EvaluateAttributeAtSample(in_TEXCOORD0.x, i1.x));
  r3.z = asfloat(i2.x) + asfloat(i1.x);
  out_SV_Target0.xyzw = asfloat(i0.xyzw) + r3.xyzw;
  return;
}
