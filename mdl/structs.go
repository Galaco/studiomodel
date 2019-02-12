package mdl

import "github.com/go-gl/mathgl/mgl32"

type Bone struct {
	NameIndex      int32
	Parent         int32
	BoneController [6]int32

	Position   mgl32.Vec3
	Quaternion mgl32.Quat
	Rotation   mgl32.Vec3

	PosScale mgl32.Vec3
	RotScale mgl32.Vec3

	PoseToBone mgl32.Mat3x4
	Alignment  mgl32.Quat

	Flags            int32
	ProcType         int32
	ProcIndex        int32
	SurfacePropIndex int32
	Contents         int32

	_ [8]int32
}

type BoneController struct {
	Bone       int32 // -1 == 0
	Type       int32 // X, Y, Z, XR, YR, ZR, M
	Start      float32
	End        float32
	Rest       int32 // byte index value at rest
	InputField int32 // 0-3 user set controller, 4 mouth
	_          [8]int32
}

type HitboxSet struct {
	NameIndex   int32
	NumHitboxes int32
	HitboxIndex int32
}

type AnimDesc struct {
	BasePtr   int32
	NameIndex int32

	Fps   float32
	Flags int32

	NumFrames     int32
	NumMovements  int32
	MovementIndex int32

	_ [6]int32

	AnimBlock int32
	AnimIndex int32

	NumIKRules           int32
	IKRuleIndex          int32
	AnimBlockIKRuleIndex int32

	NumLocalHierarchyIndex int32
	LocalHierarchyIndex    int32

	SectionIndex  int32
	SectionFrames int32

	ZeroFrameSpan  int16
	ZeroFrameCount int16
	ZeroFrameIndex int32

	ZeroFrameStallTime float32
}

type SequenceDesc struct {
	BasePtr int32

	LabelIndex int32

	ActivityNameIndex int32

	Flags int32

	Activity       int32
	ActivityWeight int32

	NumEvents  int32
	EventIndex int32

	BBMin mgl32.Vec3
	BBMax mgl32.Vec3

	NumBlends      int32
	AnimIndexIndex int32

	MovementIndex int32
	GroupSize     [2]int32
	ParamIndex    [2]int32
	ParamStart    [2]float32
	ParamEnd      [2]float32
	ParamParent   int32

	FadeinTime  float32
	FadeoutTime float32

	LocalEntryNode int32
	LocalExitNode  int32
	NodeFlags      int32

	EntryPhase float32
	ExitPhase  float32

	LastFrame float32

	NextSequence int32
	Pose         int32

	NumIKRules int32

	NumAutoLayers  int32
	AutoLayerIndex int32

	WeightListIndex int32

	PoseKeyIndex int32

	NumIKLocks  int32
	IKLockIndex int32

	KeyValueIndex int32
	KeyValueSize  int32

	CyclePoseIndex int32

	_ [7]int32
}

type Texture struct {
	NameIndex int32
	Flags     int32
	Used      int32
	_         int32

	Material       int32 // Unused by the engine. studiomdl this is a IMaterial*
	ClientMaterial int32 // Unused by the engine. studiomdl this is a void*

	_ [10]int32
}
