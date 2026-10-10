package t7dxbc

// An instruction's opcode token and extended tokens carry its controls. The
// disassembler, checked against Microsoft's on whole shader caches, writes
// them through these methods, and so does everything else that reads them:
// one decoding, one oracle.

// Opcode-token control bits.
const (
	tokSaturate    = 1 << 13
	tokTestNonZero = 1 << 18
)

// Saturate reports whether an instruction clamps its float results to [0,
// 1] (_sat).
func (in *Instruction) Saturate() bool { return in.Token&tokSaturate != 0 }

// TestsNonZero reports whether a conditional instruction (if, breakc,
// discard…) acts when any bit of its operand is set (_nz), rather than when
// none is (_z).
func (in *Instruction) TestsNonZero() bool { return in.Token&tokTestNonZero != 0 }

// ResinfoReturn is what resinfo returns: floats, the sizes' reciprocals, or
// integers.
type ResinfoReturn uint8

// The resinfo return types.
const (
	ResinfoFloat ResinfoReturn = iota
	ResinfoRcpFloat
	ResinfoUint
)

// ResinfoReturn is a resinfo instruction's return type.
func (in *Instruction) ResinfoReturn() ResinfoReturn { return ResinfoReturn(in.Token >> 11 & 3) }

// SampleinfoUint reports whether sampleinfo returns an integer (_uint)
// rather than a float.
func (in *Instruction) SampleinfoUint() bool { return in.Token>>11&1 != 0 }

// Offsets are a sample's, gather's or load's immediate texel offsets
// (_aoffimmi), each in [-8, 7].
func (in *Instruction) Offsets() [3]int32 {
	for _, x := range in.Extended {
		if x&0x3F == 1 {
			return [3]int32{signed4(x >> 9), signed4(x >> 13), signed4(x >> 17)}
		}
	}
	return [3]int32{}
}

func signed4(v uint32) int32 { return int32(v&0xF<<28) >> 28 }

// Sync flags: what a sync instruction waits for.
const (
	SyncThreads     = 1 // the group's threads (_t)
	SyncGroupShared = 2 // group-shared memory (_g)
	SyncUAVGroup    = 4 // UAV memory, for the group (_ugroup)
	SyncUAVGlobal   = 8 // UAV memory, for the device (_uglobal)
)

// SyncFlags are a sync instruction's.
func (in *Instruction) SyncFlags() uint32 { return in.Token >> 11 & 0xF }

// CustomdataICB is the class of the customdata block that holds the
// immediate constant buffer.
const CustomdataICB = 3

// CustomdataClass is a customdata block's class.
func (in *Instruction) CustomdataClass() uint32 { return in.Token >> 11 }

// InterpolationMode is a pixel shader input declaration's mode
// (D3D10_SB_INTERPOLATION_MODE): 1 constant, 2 linear, 3 linear centroid…
func (in *Instruction) InterpolationMode() uint32 { return in.Token >> 11 & 0xF }

// ResourceDimension is a resource or UAV declaration's
// D3D10_SB_RESOURCE_DIMENSION: 1 buffer, 3 texture2d, 4 texture2dms…
func (in *Instruction) ResourceDimension() uint32 { return in.Token >> 11 & 0x1F }

// Primitive is a geometry shader's input primitive or output topology
// declaration's value.
func (in *Instruction) Primitive() uint32 { return in.Token >> 11 & 0x3F }

// Global flags (D3D10_SB_GLOBAL_FLAGS), in the order the disassembler names
// them.
const (
	GlobalRefactoringAllowed = 1 << iota
	GlobalDoublePrecision
	GlobalForceEarlyDepthStencil
	GlobalRawAndStructuredBuffers
	GlobalSkipOptimization
	GlobalMinimumPrecision
	Global11_1DoubleExtensions
	Global11_1ShaderExtensions
)

// GlobalFlags are a dcl_globalFlags declaration's.
func (in *Instruction) GlobalFlags() uint32 { return in.Token >> 11 }

// UAV declaration flags.
const (
	UAVGloballyCoherent    = 1 << 16 // _glc
	UAVRasterizerOrdered   = 1 << 17 // _rov
	UAVOrderPreservingCntr = 1 << 23 // _opc
)

// UAVFlags are a UAV declaration's flags, at their bits in the opcode token.
func (in *Instruction) UAVFlags() uint32 {
	return in.Token & (UAVGloballyCoherent | UAVRasterizerOrdered | UAVOrderPreservingCntr)
}
