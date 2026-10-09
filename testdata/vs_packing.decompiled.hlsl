// Decompiled by github.com/McReaper/dxbc

cbuffer Frame : register(b0)
{
  float4x4 viewProj : packoffset(c0);
  float4 clipPlane : packoffset(c4);
  float time : packoffset(c5);
}


void main(float3 in_POSITION0 : POSITION0,
  float3 in_NORMAL0 : NORMAL0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  uint in_SV_VertexID0 : SV_VertexID0,
  uint in_SV_InstanceID0 : SV_InstanceID0,
  out float4 out_SV_Position0 : SV_Position0,
  out float2 out_TEXCOORD0 : TEXCOORD0,
  out float out_TEXCOORD2 : TEXCOORD2,
  out float out_TEXCOORD5 : TEXCOORD5,
  out nointerpolation uint out_TEXCOORD1 : TEXCOORD1,
  out nointerpolation int2 out_TEXCOORD4 : TEXCOORD4,
  out float3 out_NORMAL0 : NORMAL0,
  out float2 out_TEXCOORD3 : TEXCOORD3,
  out float out_SV_ClipDistance0 : SV_ClipDistance0,
  out float3 out_TANGENT0 : TANGENT0)
{
  float4 r0, r1;
  uint4 i0;
  
  i0.x = asuint(float(in_SV_VertexID0.x));
  i0.x = asuint(asfloat(i0.x) + time);
  r0.z = sin(asfloat(i0.x));
  r0.w = float(in_SV_InstanceID0.x);
  i0.x = asuint(r0.w + r0.w);
  r0.yw = float2(0, 1.0);
  {
    float3 dxbc_v = (float3(asfloat(i0.x), r0.y, r0.z)) + in_POSITION0.xyz;
    r0.yz = dxbc_v.yz;
    i0.x = asuint(dxbc_v.x);
  }
  r1.x = dot(float4(asfloat(i0.x), r0.y, r0.z, r0.w), float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
  r1.y = dot(float4(asfloat(i0.x), r0.y, r0.z, r0.w), float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
  r1.z = dot(float4(asfloat(i0.x), r0.y, r0.z, r0.w), float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
  r1.w = dot(float4(asfloat(i0.x), r0.y, r0.z, r0.w), float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
  out_SV_ClipDistance0.x = dot(float4(asfloat(i0.x), r0.y, r0.z, r0.w), clipPlane.xyzw);
  {
    float2 dxbc_v = (float2(asfloat(i0.x), r0.z)) * (float2(0.125, 0.125));
    r0.y = dxbc_v.y;
    i0.x = asuint(dxbc_v.x);
  }
  {
    float2 dxbc_v = floor((float2(asfloat(i0.x), r0.y)));
    r0.y = dxbc_v.y;
    i0.x = asuint(dxbc_v.x);
  }
  out_TEXCOORD4.xy = int2((float2(asfloat(i0.x), r0.y)));
  out_SV_Position0.xyzw = r1.xyzw;
  out_TEXCOORD3.xy = r1.xy / r1.ww;
  out_TEXCOORD2.x = saturate(r1.z * 0.01);
  i0.x = asuint(time * 0.25);
  out_TEXCOORD5.x = frac(asfloat(i0.x));
  out_TEXCOORD0.xy = in_TEXCOORD0.xy;
  i0.x = asuint(asint(in_SV_InstanceID0.x) << 16);
  out_TEXCOORD1.x = i0.x ^ in_SV_VertexID0.x;
  i0.x = asuint(dot(in_NORMAL0.xyz, in_NORMAL0.xyz));
  i0.x = asuint(rsqrt(asfloat(i0.x)));
  {
    float3 dxbc_v = asfloat(i0.xxx) * in_NORMAL0.xyz;
    r0.yz = dxbc_v.yz;
    i0.x = asuint(dxbc_v.x);
  }
  out_NORMAL0.xyz = asfloat(uint3(i0.x, asuint(r0.y), asuint(r0.z)));
  r1.xyz = (float3(r0.z, asfloat(i0.x), r0.y)) * (float3(1.0, 0, 0));
  out_TANGENT0.xyz = mad((float3(r0.y, r0.z, asfloat(i0.x))), (float3(0, 0, 1.0)), (-r1.xyz));
  return;
}
