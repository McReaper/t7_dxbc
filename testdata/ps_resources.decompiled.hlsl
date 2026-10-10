// Decompiled by github.com/McReaper/t7dxbc

cbuffer Material : register(b0)
{
  float4 tint : packoffset(c0);
  float cutoff : packoffset(c1);
  int2 texelShift : packoffset(c1.y);
  uint layer : packoffset(c1.w);
}

SamplerState linearSampler : register(s0);
SamplerComparisonState shadowSampler : register(s1);
Texture1D<float4> ramp : register(t0);
Texture3D<float4> volume : register(t1);
TextureCubeArray<float4> probes : register(t2);
Texture2DMS<float4, 4> msaa : register(t3);
Texture2D<float4> albedo : register(t4);
Texture2D<float> shadow : register(t5);
Texture2DArray<uint2> ids : register(t6);
Buffer<uint4> lookup : register(t7);

void main(noperspective float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  centroid float3 in_NORMAL0 : NORMAL0,
  nointerpolation uint2 in_TEXCOORD1 : TEXCOORD1,
  sample float4 in_TEXCOORD2 : TEXCOORD2,
  noperspective float in_TEXCOORD3 : TEXCOORD3,
  nointerpolation bool in_SV_IsFrontFace0 : SV_IsFrontFace0,
  out float4 out_SV_Target0 : SV_Target0,
  out uint4 out_SV_Target1 : SV_Target1,
  out float2 out_SV_Target2 : SV_Target2,
  out float oDepth : SV_Depth0)
{
  float4 r1, r3;
  uint4 i0, i1, i2, i3;
  
  i0.xyzw = asuint(albedo.Sample(linearSampler, in_TEXCOORD0.xy, int2(1, -1)).xyzw);
  {
    float4 dxbc_v = ramp.SampleLevel(linearSampler, in_TEXCOORD0.x, 2.0).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(mad(asfloat(i0.xyzw), tint.xyzw, (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w))));
  i1.xy = asuint(in_TEXCOORD0.xy);
  r1.z = in_TEXCOORD3.x;
  {
    float4 dxbc_v = volume.SampleBias(linearSampler, float3(asfloat(i1.x), asfloat(i1.y), r1.z), -0.5, int3(0, 1, -1)).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  r1.w = float(layer);
  i2.xyzw = asuint(in_NORMAL0.xyzz * (float4(0.1, 0.1, 0.1, 0.1)));
  {
    float4 dxbc_v = in_NORMAL0.zyxx * (float4(0.2, 0.2, 0.2, 0.2));
    r3.zw = dxbc_v.zw;
    i3.xy = asuint(dxbc_v.xy);
  }
  {
    float3 dxbc_v = in_NORMAL0.xyz;
    r1.z = dxbc_v.z;
    i1.xy = asuint(dxbc_v.xy);
  }
  {
    float4 dxbc_v = probes.SampleGrad(linearSampler, float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w), asfloat(i2.xyz), float3(asfloat(i3.x), asfloat(i3.y), r3.z)).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  i1.xy = asuint(int2(in_SV_Position0.xy));
  i2.x = in_TEXCOORD1.x & 3u;
  r1.zw = float2(0, 0);
  {
    float4 dxbc_v = msaa.Load(asint(i1.xy), asint(i2.x)).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  {
    float4 dxbc_v = albedo.GatherGreen(linearSampler, in_TEXCOORD0.xy, int2(-1, 0)).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  {
    float4 dxbc_v = albedo.GatherAlpha(linearSampler, in_TEXCOORD0.xy, texelShift.xy).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  {
    float4 dxbc_v = albedo.GatherRed(linearSampler, in_TEXCOORD0.xy).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)));
  {
    float4 dxbc_dims = 0;
    albedo.GetDimensions(1u, dxbc_dims.x, dxbc_dims.y, dxbc_dims.w);
    {
      float3 dxbc_v = dxbc_dims.xyw;
      r1.z = dxbc_v.z;
      i1.xy = asuint(dxbc_v.xy);
    }
  }
  {
    uint4 dxbc_dims = 0;
    volume.GetDimensions(0u, dxbc_dims.x, dxbc_dims.y, dxbc_dims.z, dxbc_dims.w);
    i2.xyzw = dxbc_dims.xyzw;
  }
  {
    uint4 dxbc_dims = 0;
    msaa.GetDimensions(dxbc_dims.x, dxbc_dims.y, dxbc_dims.w);
    dxbc_dims.w = 1u;
    i3.xy = dxbc_dims.xy;
  }
  i1.x = asuint(asfloat(i1.x) / asfloat(i1.y));
  i1.x = asuint(r1.z + asfloat(i1.x));
  i1.y = asuint(asint(i2.y) + asint(i2.x));
  i1.y = asuint(asint(i2.z) + asint(i1.y));
  i1.y = asuint(asint(i2.w) + asint(i1.y));
  i1.y = asuint(asint(i3.x) + asint(i1.y));
  i1.y = asuint(asint(i3.y) + asint(i1.y));
  i1.y = asuint(asint(i1.y) + 4);
  i1.y = asuint(float(i1.y));
  i1.x = asuint(asfloat(i1.y) + asfloat(i1.x));
  r1.w = asfloat(i0.w) + asfloat(i1.x);
  i0.w = r1.w < cutoff ? 0xffffffffu : 0u;
  if (i0.w != 0) discard;
  i0.w = asuint(shadow.SampleCmp(shadowSampler, in_TEXCOORD0.xy, in_TEXCOORD2.w));
  i1.x = asuint(asfloat(i0.w) + asfloat(i0.x));
  i0.x = asuint(shadow.GatherCmpRed(shadowSampler, in_TEXCOORD0.xy, in_TEXCOORD2.z).w);
  i1.y = asuint(asfloat(i0.x) + asfloat(i0.y));
  i0.x = asuint(float4(albedo.CalculateLevelOfDetail(linearSampler, in_TEXCOORD0.xy), albedo.CalculateLevelOfDetailUnclamped(linearSampler, in_TEXCOORD0.xy), 0, 0).x);
  i0.y = asuint(float4(albedo.CalculateLevelOfDetail(linearSampler, in_TEXCOORD0.xy), albedo.CalculateLevelOfDetailUnclamped(linearSampler, in_TEXCOORD0.xy), 0, 0).y);
  i0.x = asuint(asfloat(i0.y) + asfloat(i0.x));
  r1.z = asfloat(i0.x) + asfloat(i0.z);
  i0.xy = in_TEXCOORD1.xy;
  i0.z = layer;
  i0.w = asuint(0);
  i0.xy = ids.Load(asint(i0.xyzw)).xy;
  i0.z = i0.x & 15u;
  i2.xyz = lookup.Load(asint(i0.z)).xyz;
  out_SV_Target0.xyzw = asfloat(((in_SV_IsFrontFace0 ? 0xffffffffu : 0u).xxxx) ? (uint4(i1.x, i1.y, asuint(r1.z), asuint(r1.w))) : (asuint(-(float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w)))));
  out_SV_Target1.zw = i2.xy ^ in_TEXCOORD1.xy;
  out_SV_Target2.xy = mad((-in_SV_Position0.xy), in_TEXCOORD3.xx, in_TEXCOORD2.xy);
  i0.z = asuint(float(i2.z));
  oDepth = saturate(mad(asfloat(i0.z), 0.001, in_SV_Position0.z));
  out_SV_Target1.xy = i0.xy;
  return;
}
