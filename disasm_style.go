package t7dxbc

// The number style of each opcode's immediates: the type of its operands.
// An opcode not listed is typeless (mov, movc, the loads' addresses…).
var opStyles = map[Opcode]numberStyle{}

func init() {
	for _, op := range []Opcode{OpAdd, OpDiv, OpDp2, OpDp3, OpDp4, OpEq, OpExp, OpFrc, OpGe, OpLt, OpLog, OpMad,
		OpMax, OpMin, OpMul, OpNe, OpRoundNe, OpRoundNi, OpRoundPi, OpRoundZ, OpRsq, OpSqrt, OpSincos, OpRcp,
		OpDerivRtx, OpDerivRty, OpDerivRtxCoarse, OpDerivRtxFine, OpDerivRtyCoarse, OpDerivRtyFine, OpFtoi, OpFtou,
		OpF32tof16, OpSample, OpSampleC, OpSampleCLz, OpSampleL, OpSampleD, OpSampleB, OpGather4, OpGather4C, OpLod} {
		opStyles[op] = numFloat
	}
	for _, op := range []Opcode{OpIadd, OpIeq, OpIge, OpIlt, OpImad, OpImax, OpImin, OpImul, OpIne, OpIneg, OpIshl,
		OpIshr, OpItof, OpIbfe, OpLd, OpLdMs, OpLdUavTyped, OpEvalSnapped, OpAtomicAnd, OpAtomicOr, OpAtomicXor,
		OpAtomicCmpStore, OpAtomicIadd, OpAtomicImax, OpAtomicImin, OpAtomicUmax, OpAtomicUmin, OpImmAtomicIadd,
		OpImmAtomicAnd, OpImmAtomicOr, OpImmAtomicXor, OpImmAtomicExch, OpImmAtomicCmpExch, OpImmAtomicImax,
		OpImmAtomicImin, OpImmAtomicUmax, OpImmAtomicUmin} {
		opStyles[op] = numInt
	}
	for _, op := range []Opcode{OpUdiv, OpUlt, OpUge, OpUmul, OpUmad, OpUmax, OpUmin, OpUshr, OpUtof, OpAnd, OpOr,
		OpXor, OpNot, OpUbfe, OpBfi, OpBfrev, OpCountbits, OpFirstbitHi, OpFirstbitLo, OpFirstbitShi, OpF16tof32,
		OpCase, OpSwitch} {
		opStyles[op] = numUint
	}
}

func styleOf(op Opcode) numberStyle { return opStyles[op] }

// operandStyle is the style of one operand: a conversion's source has the
// other type than its result.
func operandStyle(op Opcode, i int, style numberStyle) numberStyle {
	return style
}
