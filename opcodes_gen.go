// Code generated from Microsoft's D3D12TokenizedProgramFormat.hpp (MIT License); see NOTICE.md. DO NOT EDIT.

package dxbc

// Opcode is an instruction's operation (D3D10_SB_OPCODE_TYPE and its later additions).
type Opcode uint32

// The opcodes, numbered as the token format numbers them.
const (
	OpAdd                                  Opcode = 0
	OpAnd                                  Opcode = 1
	OpBreak                                Opcode = 2
	OpBreakc                               Opcode = 3
	OpCall                                 Opcode = 4
	OpCallc                                Opcode = 5
	OpCase                                 Opcode = 6
	OpContinue                             Opcode = 7
	OpContinuec                            Opcode = 8
	OpCut                                  Opcode = 9
	OpDefault                              Opcode = 10
	OpDerivRtx                             Opcode = 11
	OpDerivRty                             Opcode = 12
	OpDiscard                              Opcode = 13
	OpDiv                                  Opcode = 14
	OpDp2                                  Opcode = 15
	OpDp3                                  Opcode = 16
	OpDp4                                  Opcode = 17
	OpElse                                 Opcode = 18
	OpEmit                                 Opcode = 19
	OpEmitthencut                          Opcode = 20
	OpEndif                                Opcode = 21
	OpEndloop                              Opcode = 22
	OpEndswitch                            Opcode = 23
	OpEq                                   Opcode = 24
	OpExp                                  Opcode = 25
	OpFrc                                  Opcode = 26
	OpFtoi                                 Opcode = 27
	OpFtou                                 Opcode = 28
	OpGe                                   Opcode = 29
	OpIadd                                 Opcode = 30
	OpIf                                   Opcode = 31
	OpIeq                                  Opcode = 32
	OpIge                                  Opcode = 33
	OpIlt                                  Opcode = 34
	OpImad                                 Opcode = 35
	OpImax                                 Opcode = 36
	OpImin                                 Opcode = 37
	OpImul                                 Opcode = 38
	OpIne                                  Opcode = 39
	OpIneg                                 Opcode = 40
	OpIshl                                 Opcode = 41
	OpIshr                                 Opcode = 42
	OpItof                                 Opcode = 43
	OpLabel                                Opcode = 44
	OpLd                                   Opcode = 45
	OpLdMs                                 Opcode = 46
	OpLog                                  Opcode = 47
	OpLoop                                 Opcode = 48
	OpLt                                   Opcode = 49
	OpMad                                  Opcode = 50
	OpMin                                  Opcode = 51
	OpMax                                  Opcode = 52
	OpCustomdata                           Opcode = 53
	OpMov                                  Opcode = 54
	OpMovc                                 Opcode = 55
	OpMul                                  Opcode = 56
	OpNe                                   Opcode = 57
	OpNop                                  Opcode = 58
	OpNot                                  Opcode = 59
	OpOr                                   Opcode = 60
	OpResinfo                              Opcode = 61
	OpRet                                  Opcode = 62
	OpRetc                                 Opcode = 63
	OpRoundNe                              Opcode = 64
	OpRoundNi                              Opcode = 65
	OpRoundPi                              Opcode = 66
	OpRoundZ                               Opcode = 67
	OpRsq                                  Opcode = 68
	OpSample                               Opcode = 69
	OpSampleC                              Opcode = 70
	OpSampleCLz                            Opcode = 71
	OpSampleL                              Opcode = 72
	OpSampleD                              Opcode = 73
	OpSampleB                              Opcode = 74
	OpSqrt                                 Opcode = 75
	OpSwitch                               Opcode = 76
	OpSincos                               Opcode = 77
	OpUdiv                                 Opcode = 78
	OpUlt                                  Opcode = 79
	OpUge                                  Opcode = 80
	OpUmul                                 Opcode = 81
	OpUmad                                 Opcode = 82
	OpUmax                                 Opcode = 83
	OpUmin                                 Opcode = 84
	OpUshr                                 Opcode = 85
	OpUtof                                 Opcode = 86
	OpXor                                  Opcode = 87
	OpDclResource                          Opcode = 88
	OpDclConstantBuffer                    Opcode = 89
	OpDclSampler                           Opcode = 90
	OpDclIndexRange                        Opcode = 91
	OpDclGsOutputPrimitiveTopology         Opcode = 92
	OpDclGsInputPrimitive                  Opcode = 93
	OpDclMaxOutputVertexCount              Opcode = 94
	OpDclInput                             Opcode = 95
	OpDclInputSgv                          Opcode = 96
	OpDclInputSiv                          Opcode = 97
	OpDclInputPs                           Opcode = 98
	OpDclInputPsSgv                        Opcode = 99
	OpDclInputPsSiv                        Opcode = 100
	OpDclOutput                            Opcode = 101
	OpDclOutputSgv                         Opcode = 102
	OpDclOutputSiv                         Opcode = 103
	OpDclTemps                             Opcode = 104
	OpDclIndexableTemp                     Opcode = 105
	OpDclGlobalFlags                       Opcode = 106
	opReserved107                          Opcode = 107
	OpLod                                  Opcode = 108
	OpGather4                              Opcode = 109
	OpSamplePos                            Opcode = 110
	OpSampleInfo                           Opcode = 111
	OpReserved1                            Opcode = 112
	OpHsDecls                              Opcode = 113
	OpHsControlPointPhase                  Opcode = 114
	OpHsForkPhase                          Opcode = 115
	OpHsJoinPhase                          Opcode = 116
	OpEmitStream                           Opcode = 117
	OpCutStream                            Opcode = 118
	OpEmitthencutStream                    Opcode = 119
	OpInterfaceCall                        Opcode = 120
	OpBufinfo                              Opcode = 121
	OpDerivRtxCoarse                       Opcode = 122
	OpDerivRtxFine                         Opcode = 123
	OpDerivRtyCoarse                       Opcode = 124
	OpDerivRtyFine                         Opcode = 125
	OpGather4C                             Opcode = 126
	OpGather4Po                            Opcode = 127
	OpGather4PoC                           Opcode = 128
	OpRcp                                  Opcode = 129
	OpF32tof16                             Opcode = 130
	OpF16tof32                             Opcode = 131
	OpUaddc                                Opcode = 132
	OpUsubb                                Opcode = 133
	OpCountbits                            Opcode = 134
	OpFirstbitHi                           Opcode = 135
	OpFirstbitLo                           Opcode = 136
	OpFirstbitShi                          Opcode = 137
	OpUbfe                                 Opcode = 138
	OpIbfe                                 Opcode = 139
	OpBfi                                  Opcode = 140
	OpBfrev                                Opcode = 141
	OpSwapc                                Opcode = 142
	OpDclStream                            Opcode = 143
	OpDclFunctionBody                      Opcode = 144
	OpDclFunctionTable                     Opcode = 145
	OpDclInterface                         Opcode = 146
	OpDclInputControlPointCount            Opcode = 147
	OpDclOutputControlPointCount           Opcode = 148
	OpDclTessDomain                        Opcode = 149
	OpDclTessPartitioning                  Opcode = 150
	OpDclTessOutputPrimitive               Opcode = 151
	OpDclHsMaxTessfactor                   Opcode = 152
	OpDclHsForkPhaseInstanceCount          Opcode = 153
	OpDclHsJoinPhaseInstanceCount          Opcode = 154
	OpDclThreadGroup                       Opcode = 155
	OpDclUnorderedAccessViewTyped          Opcode = 156
	OpDclUnorderedAccessViewRaw            Opcode = 157
	OpDclUnorderedAccessViewStructured     Opcode = 158
	OpDclThreadGroupSharedMemoryRaw        Opcode = 159
	OpDclThreadGroupSharedMemoryStructured Opcode = 160
	OpDclResourceRaw                       Opcode = 161
	OpDclResourceStructured                Opcode = 162
	OpLdUavTyped                           Opcode = 163
	OpStoreUavTyped                        Opcode = 164
	OpLdRaw                                Opcode = 165
	OpStoreRaw                             Opcode = 166
	OpLdStructured                         Opcode = 167
	OpStoreStructured                      Opcode = 168
	OpAtomicAnd                            Opcode = 169
	OpAtomicOr                             Opcode = 170
	OpAtomicXor                            Opcode = 171
	OpAtomicCmpStore                       Opcode = 172
	OpAtomicIadd                           Opcode = 173
	OpAtomicImax                           Opcode = 174
	OpAtomicImin                           Opcode = 175
	OpAtomicUmax                           Opcode = 176
	OpAtomicUmin                           Opcode = 177
	OpImmAtomicAlloc                       Opcode = 178
	OpImmAtomicConsume                     Opcode = 179
	OpImmAtomicIadd                        Opcode = 180
	OpImmAtomicAnd                         Opcode = 181
	OpImmAtomicOr                          Opcode = 182
	OpImmAtomicXor                         Opcode = 183
	OpImmAtomicExch                        Opcode = 184
	OpImmAtomicCmpExch                     Opcode = 185
	OpImmAtomicImax                        Opcode = 186
	OpImmAtomicImin                        Opcode = 187
	OpImmAtomicUmax                        Opcode = 188
	OpImmAtomicUmin                        Opcode = 189
	OpSync                                 Opcode = 190
	OpDadd                                 Opcode = 191
	OpDmax                                 Opcode = 192
	OpDmin                                 Opcode = 193
	OpDmul                                 Opcode = 194
	OpDeq                                  Opcode = 195
	OpDge                                  Opcode = 196
	OpDlt                                  Opcode = 197
	OpDne                                  Opcode = 198
	OpDmov                                 Opcode = 199
	OpDmovc                                Opcode = 200
	OpDtof                                 Opcode = 201
	OpFtod                                 Opcode = 202
	OpEvalSnapped                          Opcode = 203
	OpEvalSampleIndex                      Opcode = 204
	OpEvalCentroid                         Opcode = 205
	OpDclGsInstanceCount                   Opcode = 206
	OpAbort                                Opcode = 207
	OpDebugBreak                           Opcode = 208
	opReserved209                          Opcode = 209
	OpDdiv                                 Opcode = 210
	OpDfma                                 Opcode = 211
	OpDrcp                                 Opcode = 212
	OpMsad                                 Opcode = 213
	OpDtoi                                 Opcode = 214
	OpDtou                                 Opcode = 215
	OpItod                                 Opcode = 216
	OpUtod                                 Opcode = 217
	opReserved218                          Opcode = 218
	OpGather4Feedback                      Opcode = 219
	OpGather4CFeedback                     Opcode = 220
	OpGather4PoFeedback                    Opcode = 221
	OpGather4PoCFeedback                   Opcode = 222
	OpLdFeedback                           Opcode = 223
	OpLdMsFeedback                         Opcode = 224
	OpLdUavTypedFeedback                   Opcode = 225
	OpLdRawFeedback                        Opcode = 226
	OpLdStructuredFeedback                 Opcode = 227
	OpSampleLFeedback                      Opcode = 228
	OpSampleCLzFeedback                    Opcode = 229
	OpSampleClampFeedback                  Opcode = 230
	OpSampleBClampFeedback                 Opcode = 231
	OpSampleDClampFeedback                 Opcode = 232
	OpSampleCClampFeedback                 Opcode = 233
	OpCheckAccessFullyMapped               Opcode = 234
	opReserved235                          Opcode = 235
)

// opInfo is what the decoder needs of an opcode: its name in the
// disassembly, and for a declaration, how many operands precede the raw
// payload (-1: every token after the opcode is an operand).
type opInfo struct {
	name     string
	operands int
}

var opInfos = [...]opInfo{
	0:   {"add", -1},
	1:   {"and", -1},
	2:   {"break", -1},
	3:   {"breakc", -1},
	4:   {"call", -1},
	5:   {"callc", -1},
	6:   {"case", -1},
	7:   {"continue", -1},
	8:   {"continuec", -1},
	9:   {"cut", -1},
	10:  {"default", -1},
	11:  {"deriv_rtx", -1},
	12:  {"deriv_rty", -1},
	13:  {"discard", -1},
	14:  {"div", -1},
	15:  {"dp2", -1},
	16:  {"dp3", -1},
	17:  {"dp4", -1},
	18:  {"else", -1},
	19:  {"emit", -1},
	20:  {"emitthencut", -1},
	21:  {"endif", -1},
	22:  {"endloop", -1},
	23:  {"endswitch", -1},
	24:  {"eq", -1},
	25:  {"exp", -1},
	26:  {"frc", -1},
	27:  {"ftoi", -1},
	28:  {"ftou", -1},
	29:  {"ge", -1},
	30:  {"iadd", -1},
	31:  {"if", -1},
	32:  {"ieq", -1},
	33:  {"ige", -1},
	34:  {"ilt", -1},
	35:  {"imad", -1},
	36:  {"imax", -1},
	37:  {"imin", -1},
	38:  {"imul", -1},
	39:  {"ine", -1},
	40:  {"ineg", -1},
	41:  {"ishl", -1},
	42:  {"ishr", -1},
	43:  {"itof", -1},
	44:  {"label", -1},
	45:  {"ld", -1},
	46:  {"ld_ms", -1},
	47:  {"log", -1},
	48:  {"loop", -1},
	49:  {"lt", -1},
	50:  {"mad", -1},
	51:  {"min", -1},
	52:  {"max", -1},
	53:  {"customdata", -1},
	54:  {"mov", -1},
	55:  {"movc", -1},
	56:  {"mul", -1},
	57:  {"ne", -1},
	58:  {"nop", -1},
	59:  {"not", -1},
	60:  {"or", -1},
	61:  {"resinfo", -1},
	62:  {"ret", -1},
	63:  {"retc", -1},
	64:  {"round_ne", -1},
	65:  {"round_ni", -1},
	66:  {"round_pi", -1},
	67:  {"round_z", -1},
	68:  {"rsq", -1},
	69:  {"sample", -1},
	70:  {"sample_c", -1},
	71:  {"sample_c_lz", -1},
	72:  {"sample_l", -1},
	73:  {"sample_d", -1},
	74:  {"sample_b", -1},
	75:  {"sqrt", -1},
	76:  {"switch", -1},
	77:  {"sincos", -1},
	78:  {"udiv", -1},
	79:  {"ult", -1},
	80:  {"uge", -1},
	81:  {"umul", -1},
	82:  {"umad", -1},
	83:  {"umax", -1},
	84:  {"umin", -1},
	85:  {"ushr", -1},
	86:  {"utof", -1},
	87:  {"xor", -1},
	88:  {"dcl_resource", 1},
	89:  {"dcl_constant_buffer", 1},
	90:  {"dcl_sampler", 1},
	91:  {"dcl_index_range", 1},
	92:  {"dcl_gs_output_primitive_topology", 0},
	93:  {"dcl_gs_input_primitive", 0},
	94:  {"dcl_max_output_vertex_count", 0},
	95:  {"dcl_input", 1},
	96:  {"dcl_input_sgv", 1},
	97:  {"dcl_input_siv", 1},
	98:  {"dcl_input_ps", 1},
	99:  {"dcl_input_ps_sgv", 1},
	100: {"dcl_input_ps_siv", 1},
	101: {"dcl_output", 1},
	102: {"dcl_output_sgv", 1},
	103: {"dcl_output_siv", 1},
	104: {"dcl_temps", 0},
	105: {"dcl_indexable_temp", 0},
	106: {"dcl_global_flags", 0},
	107: {"reserved0", -1},
	108: {"lod", -1},
	109: {"gather4", -1},
	110: {"sample_pos", -1},
	111: {"sample_info", -1},
	112: {"reserved1", -1},
	113: {"hs_decls", 0},
	114: {"hs_control_point_phase", 0},
	115: {"hs_fork_phase", 0},
	116: {"hs_join_phase", 0},
	117: {"emit_stream", -1},
	118: {"cut_stream", -1},
	119: {"emitthencut_stream", -1},
	120: {"interface_call", -1},
	121: {"bufinfo", -1},
	122: {"deriv_rtx_coarse", -1},
	123: {"deriv_rtx_fine", -1},
	124: {"deriv_rty_coarse", -1},
	125: {"deriv_rty_fine", -1},
	126: {"gather4_c", -1},
	127: {"gather4_po", -1},
	128: {"gather4_po_c", -1},
	129: {"rcp", -1},
	130: {"f32tof16", -1},
	131: {"f16tof32", -1},
	132: {"uaddc", -1},
	133: {"usubb", -1},
	134: {"countbits", -1},
	135: {"firstbit_hi", -1},
	136: {"firstbit_lo", -1},
	137: {"firstbit_shi", -1},
	138: {"ubfe", -1},
	139: {"ibfe", -1},
	140: {"bfi", -1},
	141: {"bfrev", -1},
	142: {"swapc", -1},
	143: {"dcl_stream", 1},
	144: {"dcl_function_body", 0},
	145: {"dcl_function_table", 0},
	146: {"dcl_interface", 0},
	147: {"dcl_input_control_point_count", 0},
	148: {"dcl_output_control_point_count", 0},
	149: {"dcl_tess_domain", 0},
	150: {"dcl_tess_partitioning", 0},
	151: {"dcl_tess_output_primitive", 0},
	152: {"dcl_hs_max_tessfactor", 0},
	153: {"dcl_hs_fork_phase_instance_count", 0},
	154: {"dcl_hs_join_phase_instance_count", 0},
	155: {"dcl_thread_group", 0},
	156: {"dcl_unordered_access_view_typed", 1},
	157: {"dcl_unordered_access_view_raw", 1},
	158: {"dcl_unordered_access_view_structured", 1},
	159: {"dcl_thread_group_shared_memory_raw", 1},
	160: {"dcl_thread_group_shared_memory_structured", 1},
	161: {"dcl_resource_raw", 1},
	162: {"dcl_resource_structured", 1},
	163: {"ld_uav_typed", -1},
	164: {"store_uav_typed", -1},
	165: {"ld_raw", -1},
	166: {"store_raw", -1},
	167: {"ld_structured", -1},
	168: {"store_structured", -1},
	169: {"atomic_and", -1},
	170: {"atomic_or", -1},
	171: {"atomic_xor", -1},
	172: {"atomic_cmp_store", -1},
	173: {"atomic_iadd", -1},
	174: {"atomic_imax", -1},
	175: {"atomic_imin", -1},
	176: {"atomic_umax", -1},
	177: {"atomic_umin", -1},
	178: {"imm_atomic_alloc", -1},
	179: {"imm_atomic_consume", -1},
	180: {"imm_atomic_iadd", -1},
	181: {"imm_atomic_and", -1},
	182: {"imm_atomic_or", -1},
	183: {"imm_atomic_xor", -1},
	184: {"imm_atomic_exch", -1},
	185: {"imm_atomic_cmp_exch", -1},
	186: {"imm_atomic_imax", -1},
	187: {"imm_atomic_imin", -1},
	188: {"imm_atomic_umax", -1},
	189: {"imm_atomic_umin", -1},
	190: {"sync", -1},
	191: {"dadd", -1},
	192: {"dmax", -1},
	193: {"dmin", -1},
	194: {"dmul", -1},
	195: {"deq", -1},
	196: {"dge", -1},
	197: {"dlt", -1},
	198: {"dne", -1},
	199: {"dmov", -1},
	200: {"dmovc", -1},
	201: {"dtof", -1},
	202: {"ftod", -1},
	203: {"eval_snapped", -1},
	204: {"eval_sample_index", -1},
	205: {"eval_centroid", -1},
	206: {"dcl_gs_instance_count", 0},
	207: {"abort", -1},
	208: {"debug_break", -1},
	209: {"reserved0", -1},
	210: {"ddiv", -1},
	211: {"dfma", -1},
	212: {"drcp", -1},
	213: {"msad", -1},
	214: {"dtoi", -1},
	215: {"dtou", -1},
	216: {"itod", -1},
	217: {"utod", -1},
	218: {"reserved0", -1},
	219: {"gather4_feedback", -1},
	220: {"gather4_c_feedback", -1},
	221: {"gather4_po_feedback", -1},
	222: {"gather4_po_c_feedback", -1},
	223: {"ld_feedback", -1},
	224: {"ld_ms_feedback", -1},
	225: {"ld_uav_typed_feedback", -1},
	226: {"ld_raw_feedback", -1},
	227: {"ld_structured_feedback", -1},
	228: {"sample_l_feedback", -1},
	229: {"sample_c_lz_feedback", -1},
	230: {"sample_clamp_feedback", -1},
	231: {"sample_b_clamp_feedback", -1},
	232: {"sample_d_clamp_feedback", -1},
	233: {"sample_c_clamp_feedback", -1},
	234: {"check_access_fully_mapped", -1},
	235: {"reserved0", -1},
}
