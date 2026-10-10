// Decompiled by github.com/McReaper/t7dxbc

cbuffer Params : register(b0)
{
  float4 table[8] : packoffset(c0);
  uint mode : packoffset(c8);
  int steps : packoffset(c8.y);
  float bias : packoffset(c8.z);
  float level : packoffset(c8.w);
}

SamplerState linearSampler : register(s0);
SamplerComparisonState shadowSampler : register(s1);
Texture2D<float4> tex : register(t0);
Texture2DArray<float4> layers : register(t1);
TextureCube<float4> sky : register(t2);
Texture2D<float> depthMap : register(t3);

void ps_main(noperspective float4 in_SV_Position0 : SV_Position0,
  float3 in_TEXCOORD0 : TEXCOORD0,
  nointerpolation uint in_TEXCOORD1 : TEXCOORD1,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r0, r1, r2;
  uint4 i1;
  
  r0.xyzw = float4(0, 0, 0, 0);
  i1.x = asuint(0);
  while (true) {
    i1.y = asint(i1.x) >= steps ? 0xffffffffu : 0u;
    if (i1.y != 0) break;
    i1.y = asint(i1.x) == 3 ? 0xffffffffu : 0u;
    if (i1.y != 0) {
      i1.x = 4u;
      continue;
    }
    i1.y = asuint(asint(i1.x) + asint(in_TEXCOORD1.x));
    i1.y = i1.y & 7u;
    r2.xyzw = r0.xyzw + table[asint(i1.y)].xyzw;
    i1.y = 10.0 < r2.x ? 0xffffffffu : 0u;
    if (i1.y != 0) {
      r0.xyzw = r2.xyzw;
      break;
    }
    r0.xyzw = r2.xyzw;
    i1.x = asuint(asint(i1.x) + 1);
  }
  switch (asint(mode)) {
    case 0:
    r2.xyzw = tex.Sample(linearSampler, in_TEXCOORD0.xy).xyzw;
    {
      float4 dxbc_v = r0.xyzw + r2.xyzw;
      r1.zw = dxbc_v.zw;
      i1.xy = asuint(dxbc_v.xy);
    }
    break;
    case 1:
    r2.xyzw = tex.SampleLevel(linearSampler, in_TEXCOORD0.xy, level).xyzw;
    {
      float4 dxbc_v = r0.xyzw + r2.xyzw;
      r1.zw = dxbc_v.zw;
      i1.xy = asuint(dxbc_v.xy);
    }
    break;
    case 2:
    r2.xyzw = tex.SampleBias(linearSampler, in_TEXCOORD0.xy, bias).xyzw;
    {
      float4 dxbc_v = r0.xyzw + r2.xyzw;
      r1.zw = dxbc_v.zw;
      i1.xy = asuint(dxbc_v.xy);
    }
    break;
    default:
    r2.xy = ddx_coarse(in_TEXCOORD0.xy);
    r2.zw = ddy_coarse(in_TEXCOORD0.xy);
    r2.xyzw = tex.SampleGrad(linearSampler, in_TEXCOORD0.xy, r2.xy, r2.zw).xyzw;
    {
      float4 dxbc_v = r0.xyzw + r2.xyzw;
      r1.zw = dxbc_v.zw;
      i1.xy = asuint(dxbc_v.xy);
    }
    break;
  }
  r0.xyzw = layers.Sample(linearSampler, in_TEXCOORD0.xyz).xyzw;
  r0.xyzw = r0.xyzw + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w));
  i1.x = asuint(dot(in_TEXCOORD0.xyz, in_TEXCOORD0.xyz));
  i1.x = asuint(rsqrt(asfloat(i1.x)));
  {
    float3 dxbc_v = asfloat(i1.xxx) * in_TEXCOORD0.xyz;
    r1.z = dxbc_v.z;
    i1.xy = asuint(dxbc_v.xy);
  }
  {
    float4 dxbc_v = sky.Sample(linearSampler, float3(asfloat(i1.x), asfloat(i1.y), r1.z)).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  r0.xyzw = r0.xyzw + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w));
  i1.xy = asuint(int2(in_SV_Position0.xy));
  r1.zw = float2(0, 0);
  {
    float4 dxbc_v = tex.Load(int3(asint(i1.x), asint(i1.y), asint(r1.z))).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  r0.xyzw = r0.xyzw + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w));
  {
    float4 dxbc_v = tex.GatherRed(linearSampler, in_TEXCOORD0.xy).xyzw;
    r1.zw = dxbc_v.zw;
    i1.xy = asuint(dxbc_v.xy);
  }
  r0.xyzw = r0.xyzw + (float4(asfloat(i1.x), asfloat(i1.y), r1.z, r1.w));
  i1.x = asuint(depthMap.SampleCmpLevelZero(shadowSampler, in_TEXCOORD0.xy, in_TEXCOORD0.z));
  r0.w = r0.w * asfloat(i1.x);
  i1.x = asuint(asint(in_TEXCOORD1.x) << 3);
  i1.x = asuint(r0.x) ^ i1.x;
  i1.y = mode >> 1u;
  i1.x = i1.y | i1.x;
  i1.x = countbits(i1.x);
  i1.x = asuint(float(i1.x));
  i1.y = asuint(trunc(r0.z));
  i1.x = asuint(asfloat(i1.y) + asfloat(i1.x));
  r0.y = r0.y + asfloat(i1.x);
  i1.x = asuint(ddx_coarse(in_TEXCOORD0.x));
  i1.y = asuint(ddy_coarse(in_TEXCOORD0.x));
  i1.x = asuint(abs(asfloat(i1.y)) + abs(asfloat(i1.x)));
  out_SV_Target0.xyzw = r0.xyzw + asfloat(i1.xxxx);
  return;
}
