// Decompiled by github.com/McReaper/t7_dxbc


struct GSIn
{
  float4 SV_Position0 : SV_Position0;
  float2 TEXCOORD0 : TEXCOORD0;
  float TEXCOORD2 : TEXCOORD2;
  float2 TEXCOORD1 : TEXCOORD1;
};

struct GSOut
{
  float4 SV_Position0 : SV_Position0;
  float2 TEXCOORD0[3] : TEXCOORD0;
  float TEXCOORD3 : TEXCOORD3;
};

[maxvertexcount(3)]
void gs_main(triangle GSIn in_v[3],
  inout PointStream<GSOut> out_stream)
{
  GSOut out_v = (GSOut)0;
  float4 r0;
  
  out_v.SV_Position0.xyzw = in_v[0].SV_Position0.xyzw;
  out_v.TEXCOORD0[0].xy = in_v[0].TEXCOORD0.xy;
  out_v.TEXCOORD3.x = in_v[0].TEXCOORD2.x;
  r0.xy = in_v[1].TEXCOORD2.xx * in_v[0].TEXCOORD1.xy;
  out_v.TEXCOORD0[1].xy = r0.xy;
  r0.xy = in_v[2].TEXCOORD0.xy + (-in_v[0].TEXCOORD1.xy);
  out_v.TEXCOORD0[2].xy = r0.xy;
  out_stream.Append(out_v);
  out_v.SV_Position0.xyzw = in_v[1].SV_Position0.xyzw;
  out_v.TEXCOORD0[0].xy = in_v[1].TEXCOORD0.xy;
  out_v.TEXCOORD3.x = in_v[1].TEXCOORD2.x;
  r0.xy = in_v[2].TEXCOORD2.xx * in_v[1].TEXCOORD1.xy;
  out_v.TEXCOORD0[1].xy = r0.xy;
  r0.xy = in_v[0].TEXCOORD0.xy + (-in_v[1].TEXCOORD1.xy);
  out_v.TEXCOORD0[2].xy = r0.xy;
  out_stream.Append(out_v);
  out_v.SV_Position0.xyzw = in_v[2].SV_Position0.xyzw;
  out_v.TEXCOORD0[0].xy = in_v[2].TEXCOORD0.xy;
  out_v.TEXCOORD3.x = in_v[2].TEXCOORD2.x;
  r0.xy = in_v[0].TEXCOORD2.xx * in_v[2].TEXCOORD1.xy;
  out_v.TEXCOORD0[1].xy = r0.xy;
  r0.xy = in_v[1].TEXCOORD0.xy + (-in_v[2].TEXCOORD1.xy);
  out_v.TEXCOORD0[2].xy = r0.xy;
  out_stream.Append(out_v);
  return;
}
