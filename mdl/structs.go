package mdl

import "github.com/go-gl/mathgl/mgl32"

// Bone
type Bone struct {
	// NameIndex
	NameIndex int32
	// Parent
	Parent int32
	// BoneController
	BoneController [6]int32

	// Position
	Position mgl32.Vec3
	// Quaternion
	Quaternion mgl32.Quat
	// Rotation
	Rotation mgl32.Vec3

	// PosScale
	PosScale mgl32.Vec3
	// RotScale
	RotScale mgl32.Vec3

	// PoseToBone
	PoseToBone mgl32.Mat3x4
	// Alignment
	Alignment mgl32.Quat

	// Flags
	Flags int32
	// ProcType
	ProcType int32
	// ProcIndex
	ProcIndex int32
	// SurfacePropIndex
	SurfacePropIndex int32
	// Contents
	Contents int32

	_ [8]int32
}

// BoneController
type BoneController struct {
	// Bone
	// -1 == 0
	Bone int32
	// Type
	// X, Y, Z, XR, YR, ZR, M
	Type int32
	// Start
	Start float32
	// End
	End float32
	// Rest
	// byte index value at rest
	Rest int32
	// InputField
	// 0-3 user set controller, 4 mouth
	InputField int32
	_          [8]int32
}

// HitboxSet
type HitboxSet struct {
	// NameIndex
	NameIndex int32
	// NumHitboxes
	NumHitboxes int32
	// HitboxIndex
	HitboxIndex int32
}

// AnimDesc
type AnimDesc struct {
	// BasePtr
	BasePtr int32
	// NameIndex
	NameIndex int32

	// Fps
	Fps float32
	// Flags
	Flags int32

	// NumFrames
	NumFrames int32
	// NumMovements
	NumMovements int32
	// MovementIndex
	MovementIndex int32

	_ [6]int32

	// AnimBlock
	AnimBlock int32
	// AnimIndex
	AnimIndex int32

	// NumIKRules
	NumIKRules int32
	// IKRuleIndex
	IKRuleIndex int32
	// AnimBlockIKRuleIndex
	AnimBlockIKRuleIndex int32

	// NumLocalHierarchyIndex
	NumLocalHierarchyIndex int32
	// LocalHierarchyIndex
	LocalHierarchyIndex int32

	// SectionIndex
	SectionIndex int32
	// SectionFrames
	SectionFrames int32

	// ZeroFrameSpan
	ZeroFrameSpan int16
	// ZeroFrameCount
	ZeroFrameCount int16
	// ZeroFrameIndex
	ZeroFrameIndex int32

	// ZeroFrameStallTime
	ZeroFrameStallTime float32
}

// SequenceDesc
type SequenceDesc struct {
	// BasePtr
	BasePtr int32

	// LabelIndex
	LabelIndex int32

	// ActivityNameIndex
	ActivityNameIndex int32

	// Flags
	Flags int32

	// Activity
	Activity int32
	// ActivityWeight
	ActivityWeight int32

	// NumEvents
	NumEvents int32
	// EventIndex
	EventIndex int32

	// BBMin
	BBMin mgl32.Vec3
	// BBMax
	BBMax mgl32.Vec3

	// NumBlends
	NumBlends int32
	// AnimIndexIndex
	AnimIndexIndex int32

	// MovementIndex
	MovementIndex int32
	// GroupSize
	GroupSize [2]int32
	// ParamIndex
	ParamIndex [2]int32
	// ParamStart
	ParamStart [2]float32
	// ParamEnd
	ParamEnd [2]float32
	// ParamEnd
	ParamParent int32

	// FadeinTime
	FadeinTime float32
	// FadeoutTime
	FadeoutTime float32

	// LocalEntryNode
	LocalEntryNode int32
	// LocalExitNode
	LocalExitNode int32
	// NodeFlags
	NodeFlags int32

	// EntryPhase
	EntryPhase float32
	// ExitPhase
	ExitPhase float32

	//LastFrame
	LastFrame float32

	// NextSequence
	NextSequence int32
	// Pose
	Pose int32

	// NumIKRules
	NumIKRules int32

	// NumAutoLayers
	NumAutoLayers int32
	// AutoLayerIndex
	AutoLayerIndex int32

	// WeightListIndex
	WeightListIndex int32

	// PoseKeyIndex
	PoseKeyIndex int32

	// NumIKLocks
	NumIKLocks int32
	// IKLockIndex
	IKLockIndex int32

	// KeyValueIndex
	KeyValueIndex int32
	// KeyValueSize
	KeyValueSize int32

	// CyclePoseIndex
	CyclePoseIndex int32

	_ [7]int32
}

// Texture
type Texture struct {
	// NameIndex
	NameIndex int32
	// Flags
	Flags int32
	// Used
	Used int32
	_    int32

	// Material
	// Unused by the engine. studiomdl this is a IMaterial*
	Material int32
	// ClientMaterial
	// Unused by the engine. studiomdl this is a void*
	ClientMaterial int32

	_ [10]int32
}
