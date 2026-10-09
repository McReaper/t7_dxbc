// Decompiled by github.com/McReaper/dxbc

cbuffer Scene : register(b0)
{
  float4x4 worldViewProj : packoffset(c0);
  float4x4 world : packoffset(c4);
  float time : packoffset(c8);
}


void main(float3 in_POSITION0 : POSITION0,
  float3 in_NORMAL0 : NORMAL0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  float4 in_COLOR0 : COLOR0,
  out float4 out_SV_Position0 : SV_Position0,
  out float2 out_TEXCOORD0 : TEXCOORD0,
  out float3 out_NORMAL0 : NORMAL0,
  out float4 out_COLOR0 : COLOR0)
{
  float4 r0;
  
  r0.x = sin(time);
  r0.xyz = r0.xxx * in_NORMAL0.xyz;
  r0.xyz = mad(r0.xyz, (float3(0.05, 0.05, 0.05)), in_POSITION0.xyz);
  r0.w = 1.0;
  out_SV_Position0.x = dot(r0.xyzw, float4(worldViewProj._m00, worldViewProj._m10, worldViewProj._m20, worldViewProj._m30));
  out_SV_Position0.y = dot(r0.xyzw, float4(worldViewProj._m01, worldViewProj._m11, worldViewProj._m21, worldViewProj._m31));
  out_SV_Position0.z = dot(r0.xyzw, float4(worldViewProj._m02, worldViewProj._m12, worldViewProj._m22, worldViewProj._m32));
  out_SV_Position0.w = dot(r0.xyzw, float4(worldViewProj._m03, worldViewProj._m13, worldViewProj._m23, worldViewProj._m33));
  out_TEXCOORD0.xy = in_TEXCOORD0.xy;
  r0.x = dot(in_NORMAL0.xyz, float3(world._m00, world._m10, world._m20));
  r0.y = dot(in_NORMAL0.xyz, float3(world._m01, world._m11, world._m21));
  r0.z = dot(in_NORMAL0.xyz, float3(world._m02, world._m12, world._m22));
  r0.w = dot(r0.xyz, r0.xyz);
  r0.w = rsqrt(r0.w);
  out_NORMAL0.xyz = r0.www * r0.xyz;
  out_COLOR0.xyzw = in_COLOR0.xyzw;
  return;
}
