// Decompiled by github.com/McReaper/t7dxbc

struct Light
{
  float3 color;
  float intensity;
};

cbuffer Material : register(b0)
{
  float4 tint : packoffset(c0);
  float roughness : packoffset(c1);
  float2 uvScroll : packoffset(c1.y);
  float4x4 uvTransform : packoffset(c2);
  float4 weights[3] : packoffset(c6);
  Light key : packoffset(c9);
}

SamplerState colorSampler : register(s0);
Texture2D<float4> colorMap : register(t0);

void ps_main(float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  float3 in_NORMAL0 : NORMAL0,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r0, r1;
  uint4 i1;
  
  r0.xy = in_TEXCOORD0.xy + uvScroll.xy;
  r0.z = 1.0;
  i1.x = asuint(dot(r0.xyz, float3(uvTransform._m00, uvTransform._m10, uvTransform._m30)));
  r1.y = dot(r0.xyz, float3(uvTransform._m01, uvTransform._m11, uvTransform._m31));
  r0.xyzw = colorMap.Sample(colorSampler, float2(asfloat(i1.x), r1.y)).xyzw;
  r0.xyzw = r0.xyzw * tint.xyzw;
  i1.x = r0.w < 0.25 ? 0xffffffffu : 0u;
  if (i1.x != 0) discard;
  i1.x = asuint(dot(in_NORMAL0.xyz, in_NORMAL0.xyz));
  i1.x = asuint(rsqrt(asfloat(i1.x)));
  i1.x = asuint(saturate(asfloat(i1.x) * in_NORMAL0.z));
  r1.yzw = (float3(key.intensity, key.intensity, key.intensity)) * key.color.xyz;
  {
    float3 dxbc_v = mad(r1.yzw, asfloat(i1.xxx), weights[1].xyz);
    r1.yz = dxbc_v.yz;
    i1.x = asuint(dxbc_v.x);
  }
  r0.xyz = r0.xyz * (float3(asfloat(i1.x), r1.y, r1.z));
  i1.x = asuint((-roughness) + 1.0);
  out_SV_Target0.xyz = r0.xyz * asfloat(i1.xxx);
  out_SV_Target0.w = r0.w;
  return;
}
