// Decompiled by github.com/McReaper/t7_dxbc

cbuffer Sparks : register(b0)
{
  float4 velocity : packoffset(c0);
  float minSize : packoffset(c1);
}


struct GSIn
{
  float4 SV_Position0 : SV_Position0;
  float PSIZE0 : PSIZE0;
  nointerpolation uint TEXCOORD0 : TEXCOORD0;
};

struct GSOut
{
  float4 SV_Position0 : SV_Position0;
  float TEXCOORD0 : TEXCOORD0;
};

[maxvertexcount(2)]
void gs_main(point GSIn in_v[1],
  inout LineStream<GSOut> out_stream)
{
  GSOut out_v = (GSOut)0;
  float4 r0;
  uint4 i0;
  
  i0.x = in_v[0].PSIZE0.x < minSize ? 0xffffffffu : 0u;
  i0.y = 1u & in_v[0].TEXCOORD0.x;
  i0.y = asint(i0.y) != 0 ? 0xffffffffu : 0u;
  i0.x = i0.y | i0.x;
  if (i0.x != 0) {
    return;
  }
  out_v.SV_Position0.xyzw = in_v[0].SV_Position0.xyzw;
  out_v.TEXCOORD0.x = 1.0;
  out_stream.Append(out_v);
  {
    float4 dxbc_v = mad(velocity.xyzw, in_v[0].PSIZE0.xxxx, in_v[0].SV_Position0.xyzw);
    r0.zw = dxbc_v.zw;
    i0.xy = asuint(dxbc_v.xy);
  }
  out_v.SV_Position0.xyzw = asfloat(uint4(i0.x, i0.y, asuint(r0.z), asuint(r0.w)));
  out_v.TEXCOORD0.x = 0;
  out_stream.Append(out_v);
  return;
}
