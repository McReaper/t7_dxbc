// Decompiled by github.com/McReaper/t7_dxbc

float4 colorTint;
float2 uvScroll;
float glow;
float unused;

SamplerState colorSampler : register(s0);
Texture2D<float4> colorMap : register(t0);

void ps_main(float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r0;
  
  r0.xy = in_TEXCOORD0.xy + uvScroll.xy;
  r0.xyzw = colorMap.Sample(colorSampler, r0.xy).xyzw;
  r0.xyz = r0.xyz * colorTint.xyz;
  out_SV_Target0.w = r0.w;
  r0.w = glow + 1.0;
  out_SV_Target0.xyz = r0.www * r0.xyz;
  return;
}
