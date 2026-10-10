// Decompiled by github.com/McReaper/t7dxbc

cbuffer Blur : register(b0)
{
  float2 step : packoffset(c0);
}


void vs_main(float4 in_POSITION0 : POSITION0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  out float4 out_SV_Position0 : SV_Position0,
  out float2 out_TEXCOORD0[4] : TEXCOORD0,
  out float out_TEXCOORD4 : TEXCOORD4)
{
  float4 r0;
  
  r0.xyzw = mad(step.xyxy, (float4(-1.5, -1.5, -0.5, -0.5)), in_TEXCOORD0.xyxy);
  out_TEXCOORD0[0].xy = r0.xy;
  out_TEXCOORD0[1].xy = r0.zw;
  r0.xyzw = mad(step.xyxy, (float4(0.5, 0.5, 1.5, 1.5)), in_TEXCOORD0.xyxy);
  out_TEXCOORD0[2].xy = r0.xy;
  out_TEXCOORD0[3].xy = r0.zw;
  out_SV_Position0.xyzw = in_POSITION0.xyzw;
  out_TEXCOORD4.x = 0.25;
  return;
}
