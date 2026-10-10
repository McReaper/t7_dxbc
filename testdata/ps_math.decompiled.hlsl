// Decompiled by github.com/McReaper/t7dxbc

struct Light
{
  float3 position;
  float range;
  float4 color;
  uint flags;
};

struct Flag
{
  bool on;
  float weight;
};

cbuffer Scene : register(b0)
{
  Light lights[6] : packoffset(c0);
  row_major float3x4 bones[4] : packoffset(c18);
  float4x4 frames[3] : packoffset(c30);
  uint lightCount : packoffset(c42);
  int offset : packoffset(c42.y);
  uint mode : packoffset(c42.z);
  float gain : packoffset(c42.w);
  bool mirror : packoffset(c43);
}

StructuredBuffer<Flag> flags : register(t0);
static const uint4 icb[5] = { uint4(0x3dcccccd, 0x00000000, 0x00000000, 0x00000000), uint4(0x3e4ccccd, 0x00000000, 0x00000000, 0x00000000), uint4(0x3ecccccd, 0x00000000, 0x00000000, 0x00000000), uint4(0x3e4ccccd, 0x00000000, 0x00000000, 0x00000000), uint4(0x3dcccccd, 0x00000000, 0x00000000, 0x00000000) };

uint dxbc_bfi(uint w, uint o, uint i, uint b) { w &= 31; o &= 31; uint m = ((1u << w) - 1) << o; return ((i << o) & m) | (b & ~m); }
uint dxbc_firstbit_hi(uint x) { uint b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint2 dxbc_firstbit_hi(uint2 x) { uint2 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint3 dxbc_firstbit_hi(uint3 x) { uint3 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint4 dxbc_firstbit_hi(uint4 x) { uint4 b = firstbithigh(x); return b == 0xffffffffu ? b : 31 - b; }
uint dxbc_firstbit_shi(int x) { int b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint2 dxbc_firstbit_shi(int2 x) { int2 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint3 dxbc_firstbit_shi(int3 x) { int3 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
uint4 dxbc_firstbit_shi(int4 x) { int4 b = firstbithigh(x); return b == -1 ? 0xffffffffu : asuint(31 - b); }
int dxbc_ibfe(uint w, uint o, int s) { w &= 31; o &= 31; if (w == 0) return 0; if (w + o < 32) return (s << (32 - (w + o))) >> (32 - w); return s >> o; }
uint dxbc_ubfe(uint w, uint o, uint s) { w &= 31; o &= 31; if (w == 0) return 0; if (w + o < 32) return (s << (32 - (w + o))) >> (32 - w); return s >> o; }

void main(float4 in_SV_Position0 : SV_Position0,
  float2 in_TEXCOORD0 : TEXCOORD0,
  nointerpolation uint4 in_TEXCOORD1 : TEXCOORD1,
  nointerpolation int4 in_TEXCOORD2 : TEXCOORD2,
  out float4 out_SV_Target0 : SV_Target0)
{
  float4 r4, r5, r6, r7;
  uint4 i0, i1, i2, i3, i6;
  uint4 x0[8];
  
  i0.x = dxbc_ubfe(6u, 5u, in_TEXCOORD1.x);
  i0.y = asuint(dxbc_ibfe(12u, 13u, in_TEXCOORD2.x));
  i0.z = dxbc_bfi(8u, 8u, in_TEXCOORD1.y, in_TEXCOORD1.x);
  i0.w = dxbc_firstbit_hi(in_TEXCOORD1.x);
  i0.w = asuint((-asint(i0.w)) + 31);
  i0.w = in_TEXCOORD1.x ? i0.w : 0xffffffffu;
  i1.x = firstbitlow(in_TEXCOORD1.x);
  i1.y = dxbc_firstbit_shi(in_TEXCOORD2.y);
  i1.z = asint(i1.y) == (-1) ? 0xffffffffu : 0u;
  i1.y = asuint((-asint(i1.y)) + 31);
  i1.w = reversebits(in_TEXCOORD1.z);
  i2.x = countbits(in_TEXCOORD1.w);
  i1.w = i1.w ^ i2.x;
  i2.x = in_TEXCOORD1.w | 1u;
  {
    uint dxbc_first = in_TEXCOORD1.z / i2.x;
    i3.x = in_TEXCOORD1.z % i2.x;
    i2.x = dxbc_first;
  }
  i2.y = asuint(in_TEXCOORD2.z * in_TEXCOORD2.w + offset);
  i2.z = max(in_TEXCOORD1.y, in_TEXCOORD1.x);
  i2.w = min(in_TEXCOORD1.w, in_TEXCOORD1.z);
  i2.z = asuint((-asint(i2.w)) + asint(i2.z));
  i2.w = asuint(max(in_TEXCOORD2.y, in_TEXCOORD2.x));
  i3.y = asuint(min((-in_TEXCOORD2.w), in_TEXCOORD2.z));
  i2.w = asuint(asint(i2.w) + (-asint(i3.y)));
  i3.yz = f32tof16(in_TEXCOORD0.xy);
  i3.y = asuint(asint(i3.z) * 65536 + asint(i3.y));
  r4.x = f16tof32(i3.y);
  i3.y = dxbc_ubfe(16u, 0u, i3.z);
  r4.y = f16tof32(i3.y);
  r5.xyzw = in_TEXCOORD0.xyxy * (float4(6.2831855, 7.5, 3.3, 2.2));
  i3.y = asuint(round(r5.y));
  i3.z = asuint(floor(r5.z));
  i3.y = asuint(asfloat(i3.z) + asfloat(i3.y));
  i3.z = asuint(ceil(r5.w));
  i3.y = asuint(asfloat(i3.z) + asfloat(i3.y));
  i3.z = asuint(in_TEXCOORD0.x * (-4.4));
  i3.z = asuint(trunc(asfloat(i3.z)));
  r4.w = asfloat(i3.z) + asfloat(i3.y);
  x0[0].x = asuint(in_TEXCOORD0.y);
  i3.y = asuint(in_TEXCOORD0.y + in_TEXCOORD0.x);
  x0[1].x = i3.y;
  i3.y = asuint(mad(in_TEXCOORD0.x, 2.0, in_TEXCOORD0.y));
  x0[2].x = i3.y;
  i3.yzw = asuint(mad(in_TEXCOORD0.xxx, (float3(3.0, 4.0, 5.0)), in_TEXCOORD0.yyy));
  x0[3].x = i3.y;
  x0[4].x = i3.z;
  x0[5].x = i3.w;
  i3.yz = asuint(mad(in_TEXCOORD0.xx, (float2(6.0, 7.0)), in_TEXCOORD0.yy));
  x0[6].x = i3.y;
  x0[7].x = i3.z;
  i3.y = asuint(asint(in_TEXCOORD1.x) + asint(mode));
  i3.y = i3.y & 7u;
  i3.y = x0[asint(i3.y)].x;
  i3.zw = in_TEXCOORD1.yy % (uint2(5u, 3u));
  r4.z = asfloat(i3.y) + asfloat(icb[asint(i3.z)].x);
  r6.xy = in_TEXCOORD0.xy;
  r6.z = 0;
  r5.yzw = float3(0, 0, 0);
  i3.y = asuint(0);
  while (true) {
    i3.z = i3.y >= lightCount ? 0xffffffffu : 0u;
    if (i3.z != 0) break;
    i3.z = asuint(asint(i3.y) + asint(in_TEXCOORD1.x));
    i3.z = i3.z % 6u;
    i3.z = asuint(asint(i3.z) * 3);
    i6.w = 1u & (lights[i3.z / 3u].flags);
    if (i6.w == 0) {
      i6.w = asuint(asint(i3.y) + 1);
      i3.y = i6.w;
      continue;
    }
    r7.xyz = (-r6.xyz) + (lights[i3.z / 3u].position.xyz);
    i6.w = asuint(dot(r7.xyz, r7.xyz));
    i6.w = asuint(sqrt(asfloat(i6.w)));
    i6.w = asuint(asfloat(i6.w) / (lights[i3.z / 3u].range));
    i6.w = asuint(saturate((-asfloat(i6.w)) + 1.0));
    r5.yzw = mad((lights[i3.z / 3u].color.xyz), asfloat(i6.www), r5.yzw);
    i3.y = asuint(asint(i3.y) + 1);
  }
  i3.yz = in_TEXCOORD1.xx & (uint2(3u, 7u));
  i3.y = asuint(asint(i3.y) * 3);
  r6.xy = in_TEXCOORD0.xy;
  {
    float2 dxbc_v = float2(1.0, 1.0);
    r6.z = dxbc_v.x;
    i6.w = asuint(dxbc_v.y);
  }
  r7.x = dot(float4(bones[i3.y / 3u]._m00, bones[i3.y / 3u]._m01, bones[i3.y / 3u]._m02, bones[i3.y / 3u]._m03), float4(r6.x, r6.y, asfloat(i6.w), asfloat(i6.w)));
  r7.y = dot(float4(bones[i3.y / 3u]._m10, bones[i3.y / 3u]._m11, bones[i3.y / 3u]._m12, bones[i3.y / 3u]._m13), float4(r6.x, r6.y, asfloat(i6.w), asfloat(i6.w)));
  r7.z = dot(float4(bones[i3.y / 3u]._m20, bones[i3.y / 3u]._m21, bones[i3.y / 3u]._m22, bones[i3.y / 3u]._m23), float4(r6.x, r6.y, r6.z, asfloat(i6.w)));
  i3.y = asuint(asint(i3.w) << 2);
  i3.w = asuint(r7.y * (frames[i3.y / 4u]._m31));
  i3.w = asuint(mad((frames[i3.y / 4u]._m30), r7.x, asfloat(i3.w)));
  i3.w = asuint(mad((frames[i3.y / 4u]._m32), r7.z, asfloat(i3.w)));
  i6.w = asuint(asfloat(i3.w) + (frames[i3.y / 4u]._m33));
  r6.xyz = r5.yzw + r7.xyz;
  switch (asint(mode)) {
    case 0:
    case 1:
    i3.y = asuint(sin(r5.x));
    r6.x = asfloat(i3.y) + r6.x;
    break;
    case 2:
    i3.y = asuint(cos(r5.x));
    r6.y = mad(asfloat(i3.y), gain, r6.y);
    break;
    case 5:
    r6.z = (-r4.z) + r6.z;
    break;
    default:
    i6.w = asuint(r4.w * asfloat(i6.w));
    break;
  }
  i0.w = asuint(asint(i0.w) + asint(i1.x));
  r5.xzw = float3(i0.xzw);
  r5.y = float(asint(i0.y));
  i0.y = asuint(float(i1.w));
  i0.z = asuint(float(i2.x));
  i0.w = asuint(float(i3.x));
  i1.x = asuint(float(asint(i1.y)));
  i0.x = i1.z ? asuint(-1.0) : i1.x;
  i0.xyzw = asuint(asfloat(i0.xyzw) * (float4(0.0001, 0.0001, 0.0001, 0.0001)));
  i0.xyzw = asuint(mad(r5.xyzw, (float4(0.001, 0.001, 0.001, 0.001)), asfloat(i0.xyzw)));
  i0.xyzw = asuint(asfloat(i0.xyzw) + (float4(r6.x, r6.y, r6.z, asfloat(i6.w))));
  i1.y = asuint(float(i2.z));
  i1.xz = asuint(float2(asint(i2.yw)));
  i1.w = asuint(0);
  i1.xyzw = asuint(mad(asfloat(i1.xyzw), (float4(1e-05, 1e-05, 1e-05, 1e-05)), r4.xyzw));
  i0.xyzw = asuint(asfloat(i0.xyzw) + asfloat(i1.xyzw));
  i1.xy = uint2((flags[i3.z].on ? 0xffffffffu : 0u), asuint(flags[i3.z].weight));
  i0.xy = (uint2((mirror ? 0xffffffffu : 0u), (mirror ? 0xffffffffu : 0u))) ? i0.yx : i0.xy;
  i2.xyzw = asuint(asfloat(i1.yyyy) * asfloat(i0.xyzw));
  out_SV_Target0.xyzw = asfloat(i1.xxxx ? i2.xyzw : i0.xyzw);
  return;
}
