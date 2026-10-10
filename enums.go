package t7dxbc

// VarClass is D3D_SHADER_VARIABLE_CLASS.
type VarClass uint16

// The variable classes.
const (
	ClassScalar VarClass = iota
	ClassVector
	ClassMatrixRows
	ClassMatrixColumns
	ClassObject
	ClassStruct
	ClassInterfaceClass
	ClassInterfacePointer
)

// VarType is D3D_SHADER_VARIABLE_TYPE.
type VarType uint16

// The variable base types a constant buffer or a structured buffer element
// holds.
const (
	TypeVoid   VarType = 0
	TypeBool   VarType = 1
	TypeInt    VarType = 2
	TypeFloat  VarType = 3
	TypeUint   VarType = 19
	TypeUint8  VarType = 20
	TypeDouble VarType = 39
	// The minimum-precision types (Shader Model 5 on Windows 8).
	TypeMin8Float  VarType = 50
	TypeMin10Float VarType = 51
	TypeMin16Float VarType = 52
	TypeMin12Int   VarType = 53
	TypeMin16Int   VarType = 54
	TypeMin16Uint  VarType = 55
)

// String is the base type's HLSL name.
func (t VarType) String() string {
	switch t {
	case TypeVoid:
		return "void"
	case TypeBool:
		return "bool"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeUint:
		return "uint"
	case TypeUint8:
		return "uint8_t"
	case TypeDouble:
		return "double"
	case TypeMin8Float:
		return "min8float"
	case TypeMin10Float:
		return "min10float"
	case TypeMin16Float:
		return "min16float"
	case TypeMin12Int:
		return "min12int"
	case TypeMin16Int:
		return "min16int"
	case TypeMin16Uint:
		return "min16uint"
	}
	return "unknown"
}

// InputType is D3D_SHADER_INPUT_TYPE: what a Binding binds.
type InputType uint32

// The binding kinds.
const (
	InputCBuffer InputType = iota
	InputTBuffer
	InputTexture
	InputSampler
	InputUAVTyped
	InputStructured
	InputUAVStructured
	InputByteAddress
	InputUAVByteAddress
	InputUAVAppendStructured
	InputUAVConsumeStructured
	InputUAVStructuredWithCounter
)

// IsUAV reports whether the binding is an unordered-access view (a u#
// register).
func (t InputType) IsUAV() bool {
	switch t {
	case InputUAVTyped, InputUAVStructured, InputUAVByteAddress, InputUAVAppendStructured,
		InputUAVConsumeStructured, InputUAVStructuredWithCounter:
		return true
	}
	return false
}

// ReturnType is D3D_RESOURCE_RETURN_TYPE: what a typed texture or buffer
// returns.
type ReturnType uint32

// The return types.
const (
	ReturnUnorm ReturnType = 1 + iota
	ReturnSnorm
	ReturnSint
	ReturnUint
	ReturnFloat
	ReturnMixed
	ReturnDouble
	ReturnContinued
)

// String is the return type's HLSL name.
func (t ReturnType) String() string {
	switch t {
	case ReturnUnorm:
		return "unorm float"
	case ReturnSnorm:
		return "snorm float"
	case ReturnSint:
		return "int"
	case ReturnUint:
		return "uint"
	case ReturnFloat:
		return "float"
	case ReturnDouble:
		return "double"
	}
	return "float"
}

// Dimension is D3D_SRV_DIMENSION: the shape of a texture or buffer view.
type Dimension uint32

// The view dimensions.
const (
	DimUnknown Dimension = iota
	DimBuffer
	DimTexture1D
	DimTexture1DArray
	DimTexture2D
	DimTexture2DArray
	DimTexture2DMS
	DimTexture2DMSArray
	DimTexture3D
	DimTextureCube
	DimTextureCubeArray
	DimBufferEx
)

// Binding flags (D3D_SHADER_INPUT_FLAGS).
const (
	BindUserPacked         = 1
	BindComparisonSampler  = 2
	bindComponentCountBits = 0xC // (count-1) << 2 for a texture's return vector
)

// Components is how many components a texture or typed buffer returns
// (float4: 4).
func (b Binding) Components() int { return int(b.Flags&bindComponentCountBits)>>2 + 1 }

// SystemValue is D3D_NAME: the meaning the pipeline gives a signature
// element.
type SystemValue uint32

// The system values a signature element can carry.
const (
	SVUndefined SystemValue = iota
	SVPosition
	SVClipDistance
	SVCullDistance
	SVRenderTargetArrayIndex
	SVViewportArrayIndex
	SVVertexID
	SVPrimitiveID
	SVInstanceID
	SVIsFrontFace
	SVSampleIndex
	SVTarget            SystemValue = 64
	SVDepth             SystemValue = 65
	SVCoverage          SystemValue = 66
	SVDepthGreaterEqual SystemValue = 67
	SVDepthLessEqual    SystemValue = 68
	SVStencilRef        SystemValue = 69
	SVInnerCoverage     SystemValue = 70
)

// ComponentType is D3D_REGISTER_COMPONENT_TYPE: the scalar type of a
// signature element.
type ComponentType uint32

// The component types.
const (
	ComponentUnknown ComponentType = iota
	ComponentUint
	ComponentSint
	ComponentFloat
)

// String is the component type's HLSL name.
func (t ComponentType) String() string {
	switch t {
	case ComponentUint:
		return "uint"
	case ComponentSint:
		return "int"
	case ComponentFloat:
		return "float"
	}
	return "unknown"
}
