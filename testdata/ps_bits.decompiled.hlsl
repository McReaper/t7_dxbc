// Decompiled by github.com/McReaper/dxbc

cbuffer Params : register(b0)
{
  float4 fv : packoffset(c0);
  float fs : packoffset(c1);
}


uint dxbc_firstbit_hi(uint x) { uint b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint2 dxbc_firstbit_hi(uint2 x) { uint2 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint3 dxbc_firstbit_hi(uint3 x) { uint3 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint4 dxbc_firstbit_hi(uint4 x) { uint4 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint dxbc_firstbit_shi(int x) { int b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint2 dxbc_firstbit_shi(int2 x) { int2 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint3 dxbc_firstbit_shi(int3 x) { int3 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint4 dxbc_firstbit_shi(int4 x) { int4 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }

void main(float4 in_TEXCOORD0 : TEXCOORD0,
  nointerpolation uint4 in_TEXCOORD1 : TEXCOORD1,
  nointerpolation int4 in_TEXCOORD2 : TEXCOORD2,
  out float4 out_SV_Target0 : SV_Target0,
  out uint4 out_SV_Target1 : SV_Target1)
{
  uint4 i0, i1, i2;
  
  out_SV_Target0.xyzw = in_TEXCOORD0.xyzw;
  i0.xyz = asuint(mad((float3(fs, fs, fs)), (float3(fs, fs, fs)), in_TEXCOORD0.yzz));
  i0.x = asuint(dot(-in_TEXCOORD0.xzw, asfloat(i0.xyz)));
  i0.xy = in_TEXCOORD0.yx < asfloat(i0.xx) ? 0xffffffffu : 0u;
  i0.x = i0.y | i0.x;
  i0.y = dxbc_firstbit_hi(in_TEXCOORD1.y);
  i0.y = asuint((-asint(i0.y)) + 31);
  i1.x = in_TEXCOORD1.y ? i0.y : 0xffffffffu;
  i1.yz = in_TEXCOORD1.yz;
  i2.x = in_TEXCOORD1.x;
  i2.yz = uint2(0xb0000000u, 0x80000000u);
  i0.xyz = i0.xxx ? i1.xyz : i2.xyz;
  i1.xyzw = asuint(int4(fv.xyzw));
  i1.xyzw = asuint(asint(i1.xyzw) + in_TEXCOORD2.xyzw);
  i1.xyzw = dxbc_firstbit_shi(asint(i1.xyzw));
  i2.xyzw = asint(i1.xyzw) == (int4(-1, -1, -1, -1)) ? 0xffffffffu : 0u;
  i1.xyzw = asuint((-asint(i1.xyzw)) + (int4(31, 31, 31, 31)));
  i1.xyzw = i2.xyzw ? (uint4(0xffffffffu, 0xffffffffu, 0xffffffffu, 0xffffffffu)) : i1.xyzw;
  i0.w = in_TEXCOORD1.w;
  out_SV_Target1.xyzw = i0.xyzw ^ i1.xyzw;
  return;
}
