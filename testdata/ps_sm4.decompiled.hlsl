// Decompiled by github.com/McReaper/t7_dxbc

struct lights_t
{
  float3 position;
  float range;
};

struct a_b_t
{
  float2 uv;
};

struct a_t
{
  a_b_t b;
  float w;
};

struct a_b_t2
{
  float4 tint;
};

struct flags_t
{
  bool on;
  float weight;
};

cbuffer Scene : register(b0)
{
  lights_t lights[2] : packoffset(c0);
  a_t a : packoffset(c2);
  a_b_t2 a_b : packoffset(c3);
}

StructuredBuffer<flags_t> flags : register(t0);

void ps_main(noperspective float4 in_SV_Position0 : SV_Position0,
  nointerpolation uint in_TEXCOORD0 : TEXCOORD0,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r0, r1;
  uint4 i0;
  
  i0.xy = in_TEXCOORD0.xx & (uint2(3u, 1u));
  {
    float2 dxbc_v = a.b.uv.xy + lights[asint(i0.y)].position.xy;
    r0.w = dxbc_v.y;
    i0.z = asuint(dxbc_v.x);
  }
  {
    float2 dxbc_v = (float2(asfloat(i0.z), r0.w)) + (-in_SV_Position0.xy);
    r0.w = dxbc_v.y;
    i0.z = asuint(dxbc_v.x);
  }
  i0.z = asuint(dot(float2(asfloat(i0.z), r0.w), float2(asfloat(i0.z), r0.w)));
  i0.z = asuint(sqrt(asfloat(i0.z)));
  i0.y = asuint(asfloat(i0.z) / lights[asint(i0.y)].range);
  i0.xz = uint2((flags[i0.x].on ? 0xffffffffu : 0u), asuint(flags[i0.x].weight));
  r1.z = a.w * a_b.tint.z;
  r1.w = a_b.tint.w;
  r1.xy = asfloat(i0.yz) * a_b.tint.xy;
  out_SV_Target0.xyzw = asfloat(i0.xxxx & asuint(r1.xyzw));
  return;
}
