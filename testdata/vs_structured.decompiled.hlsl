// Decompiled by github.com/McReaper/t7dxbc

struct Instance
{
  float4x3 transform;
  float4 color;
  uint flags;
};

cbuffer Camera : register(b0)
{
  float4x4 viewProj : packoffset(c0);
}

StructuredBuffer<Instance> instances : register(t0);
ByteAddressBuffer extra : register(t1);

void main(float3 in_POSITION0 : POSITION0,
  uint in_SV_InstanceID0 : SV_InstanceID0,
  out float4 out_SV_Position0 : SV_Position0,
  out float4 out_COLOR0 : COLOR0)
{
  float4 r1;
  uint4 i0, i1, i2;
  
  i0.xyzw = uint4(asuint(instances[in_SV_InstanceID0.x].transform._m00), asuint(instances[in_SV_InstanceID0.x].transform._m10), asuint(instances[in_SV_InstanceID0.x].transform._m20), asuint(instances[in_SV_InstanceID0.x].transform._m30));
  {
    float3 dxbc_v = in_POSITION0.xyz;
    r1.yz = dxbc_v.yz;
    i1.x = asuint(dxbc_v.x);
  }
  r1.w = 1.0;
  i0.x = asuint(dot(float4(asfloat(i1.x), r1.y, r1.z, r1.w), asfloat(i0.xyzw)));
  i2.xyzw = uint4(asuint(instances[in_SV_InstanceID0.x].transform._m01), asuint(instances[in_SV_InstanceID0.x].transform._m11), asuint(instances[in_SV_InstanceID0.x].transform._m21), asuint(instances[in_SV_InstanceID0.x].transform._m31));
  i0.y = asuint(dot(float4(asfloat(i1.x), r1.y, r1.z, r1.w), asfloat(i2.xyzw)));
  i2.xyzw = uint4(asuint(instances[in_SV_InstanceID0.x].transform._m02), asuint(instances[in_SV_InstanceID0.x].transform._m12), asuint(instances[in_SV_InstanceID0.x].transform._m22), asuint(instances[in_SV_InstanceID0.x].transform._m32));
  i0.z = asuint(dot(float4(asfloat(i1.x), r1.y, r1.z, r1.w), asfloat(i2.xyzw)));
  i0.w = asuint(1.0);
  out_SV_Position0.x = dot(asfloat(i0.xyzw), float4(viewProj._m00, viewProj._m10, viewProj._m20, viewProj._m30));
  out_SV_Position0.y = dot(asfloat(i0.xyzw), float4(viewProj._m01, viewProj._m11, viewProj._m21, viewProj._m31));
  out_SV_Position0.z = dot(asfloat(i0.xyzw), float4(viewProj._m02, viewProj._m12, viewProj._m22, viewProj._m32));
  out_SV_Position0.w = dot(asfloat(i0.xyzw), float4(viewProj._m03, viewProj._m13, viewProj._m23, viewProj._m33));
  i0.x = asuint(asint(in_SV_InstanceID0.x) << 2);
  i0.x = extra.Load(i0.x);
  {
    float4 dxbc_v = instances[in_SV_InstanceID0.x].color.xyzw;
    r1.yzw = dxbc_v.yzw;
    i1.x = asuint(dxbc_v.x);
  }
  i0.xyzw = asuint(asfloat(i0.xxxx) * (float4(asfloat(i1.x), r1.y, r1.z, r1.w)));
  i1.x = instances[in_SV_InstanceID0.x].flags;
  i1.x = i1.x & 1u;
  i1.x = asuint(i1.x ? 0.5 : 1.0);
  out_COLOR0.xyzw = asfloat(i0.xyzw) * asfloat(i1.xxxx);
  return;
}
