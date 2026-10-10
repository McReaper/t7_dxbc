// Decompiled by github.com/McReaper/t7_dxbc

cbuffer Extrude : register(b0)
{
  float4x4 viewProj : packoffset(c0);
  float depth : packoffset(c4);
  uint sides : packoffset(c4.y);
}


struct GSIn
{
  float4 SV_Position0 : SV_Position0;
  float3 NORMAL0 : NORMAL0;
  float2 TEXCOORD0 : TEXCOORD0;
};

struct GSOut
{
  float4 SV_Position0 : SV_Position0;
  float3 NORMAL0 : NORMAL0;
  float2 TEXCOORD0 : TEXCOORD0;
  nointerpolation uint TEXCOORD1 : TEXCOORD1;
};

[maxvertexcount(18)]
[instance(2)]
void gs_main(triangle GSIn in_v[3],
  inout TriangleStream<GSOut> out_stream,
  uint vPrim : SV_PrimitiveID,
  uint vGSInstanceID : SV_GSInstanceID)
{
  GSOut out_v = (GSOut)0;
  float4 r0, r2, r3, r4, r5;
  uint4 i0, i1, i3;
  
  r0.xyz = (-in_v[0].SV_Position0.zxy) + in_v[1].SV_Position0.zxy;
  i1.xyz = asuint((-in_v[0].SV_Position0.yzx) + in_v[2].SV_Position0.yzx);
  r2.xyz = r0.xyz * asfloat(i1.xyz);
  r0.xyz = mad(r0.zxy, asfloat(i1.yzx), (-r2.xyz));
  i0.w = asuint(dot(r0.xyz, r0.xyz));
  i0.w = asuint(rsqrt(asfloat(i0.w)));
  r0.xyz = asfloat(i0.www) * r0.xyz;
  i0.w = asuint(asint(vGSInstanceID) + 1);
  i0.w = asuint(float(i0.w));
  i0.w = asuint(asfloat(i0.w) * depth);
  i1.x = min(sides, 3u);
  i1.y = asuint(asint(vPrim) << 2);
  i1.z = asuint(asint(vGSInstanceID) << 6);
  r2.xyz = asfloat(i0.www) * r0.xyz;
  r2.w = 0;
  i0.w = asuint(0);
  while (true) {
    i1.w = i0.w >= i1.x ? 0xffffffffu : 0u;
    if (i1.w != 0) break;
    i1.w = asuint(asint(i0.w) + 1);
    i3.x = i1.w % 3u;
    i3.y = asuint(asint(i0.w) + asint(i1.y));
    i3.y = asuint(asint(i1.z) + asint(i3.y));
    r3.z = dot(in_v[asint(i0.w)].SV_Position0.xyzw, float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
    r3.w = dot(in_v[asint(i0.w)].SV_Position0.xyzw, float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
    r4.x = dot(in_v[asint(i0.w)].SV_Position0.xyzw, float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
    r4.y = dot(in_v[asint(i0.w)].SV_Position0.xyzw, float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
    out_v.SV_Position0.x = r3.z;
    out_v.SV_Position0.y = r3.w;
    out_v.SV_Position0.z = r4.x;
    out_v.SV_Position0.w = r4.y;
    out_v.NORMAL0.xyz = r0.xyz;
    out_v.TEXCOORD0.xy = in_v[asint(i0.w)].TEXCOORD0.xy;
    out_v.TEXCOORD1.x = i3.y;
    out_stream.Append(out_v);
    r3.z = dot(in_v[asint(i3.x)].SV_Position0.xyzw, float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
    r3.w = dot(in_v[asint(i3.x)].SV_Position0.xyzw, float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
    r4.x = dot(in_v[asint(i3.x)].SV_Position0.xyzw, float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
    r4.y = dot(in_v[asint(i3.x)].SV_Position0.xyzw, float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
    out_v.SV_Position0.x = r3.z;
    out_v.SV_Position0.y = r3.w;
    out_v.SV_Position0.z = r4.x;
    out_v.SV_Position0.w = r4.y;
    out_v.NORMAL0.xyz = r0.xyz;
    out_v.TEXCOORD0.xy = in_v[asint(i3.x)].TEXCOORD0.xy;
    out_v.TEXCOORD1.x = i3.y;
    out_stream.Append(out_v);
    r4.xyzw = r2.xyzw + in_v[asint(i0.w)].SV_Position0.xyzw;
    r3.z = dot(r4.xyzw, float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
    r3.w = dot(r4.xyzw, float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
    r5.x = dot(r4.xyzw, float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
    r4.x = dot(r4.xyzw, float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
    r4.yz = (float2(0.5, 0.5)) * in_v[asint(i0.w)].TEXCOORD0.xy;
    out_v.SV_Position0.x = r3.z;
    out_v.SV_Position0.y = r3.w;
    out_v.SV_Position0.z = r5.x;
    out_v.SV_Position0.w = r4.x;
    out_v.NORMAL0.xyz = r0.xyz;
    out_v.TEXCOORD0.xy = r4.yz;
    out_v.TEXCOORD1.x = i3.y;
    out_stream.Append(out_v);
    r4.xyzw = r2.xyzw + in_v[asint(i3.x)].SV_Position0.xyzw;
    r3.z = dot(r4.xyzw, float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
    r3.w = dot(r4.xyzw, float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
    r5.x = dot(r4.xyzw, float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
    r4.x = dot(r4.xyzw, float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
    r4.yz = (float2(0.5, 0.5)) * in_v[asint(i3.x)].TEXCOORD0.xy;
    out_v.SV_Position0.x = r3.z;
    out_v.SV_Position0.y = r3.w;
    out_v.SV_Position0.z = r5.x;
    out_v.SV_Position0.w = r4.x;
    out_v.NORMAL0.xyz = r0.xyz;
    out_v.TEXCOORD0.xy = r4.yz;
    out_v.TEXCOORD1.x = i3.y;
    out_stream.Append(out_v);
    out_stream.RestartStrip();
    i0.w = i1.w;
  }
  return;
}
