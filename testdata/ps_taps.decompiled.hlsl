// Decompiled by github.com/McReaper/t7_dxbc

SamplerState linearClamp : register(s0);
Texture2D<float4> image : register(t0);

void ps_main(float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0[4] : TEXCOORD0,
  float in_TEXCOORD4 : TEXCOORD4,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r0, r1;
  
  r0.xyzw = image.Sample(linearClamp, in_TEXCOORD0[0].xy).xyzw;
  r1.xyzw = image.Sample(linearClamp, in_TEXCOORD0[1].xy).xyzw;
  r0.xyzw = r0.xyzw + r1.xyzw;
  r1.xyzw = image.Sample(linearClamp, in_TEXCOORD0[3].xy).xyzw;
  r0.xyzw = r0.xyzw + r1.xyzw;
  out_SV_Target0.xyzw = r0.xyzw * in_TEXCOORD4.xxxx;
  return;
}
