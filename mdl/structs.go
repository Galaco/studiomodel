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

// BodyPart represents a bodygroup (e.g., "head", "body", "arms")
// Corresponds to mstudiobodyparts_t in studio.h
type BodyPart struct {
	// NameIndex - offset to bodypart name string (relative to this struct)
	NameIndex int32
	// NumModels - number of models in this bodypart
	NumModels int32
	// Base - used for bodygroup value calculation
	Base int32
	// ModelIndex - byte offset from start of this struct to first Model
	ModelIndex int32
}

// Model represents a model within a bodypart (e.g., different head variations)
// Corresponds to mstudiomodel_t in studio.h
type Model struct {
	// Name - 64-byte null-padded model name
	Name [64]byte
	// Type - model type (unused in most cases)
	Type int32
	// BoundingRadius - bounding sphere radius
	BoundingRadius float32

	// NumMeshes - number of meshes in this model
	NumMeshes int32
	// MeshIndex - byte offset from start of this struct to first Mesh
	MeshIndex int32

	// NumVertices - total unique vertices in this model
	NumVertices int32
	// VertexIndex - offset to vertex data (relative to model start)
	VertexIndex int32
	// VertexInfoIndex - offset to vertex bone info
	VertexInfoIndex int32

	// NumNormals - total normals
	NumNormals int32
	// NormalIndex - offset to normal data
	NormalIndex int32
	// NormalInfoIndex - offset to normal bone info
	NormalInfoIndex int32

	// NumGroups - number of deformation groups
	NumGroups int32
	// GroupIndex - offset to groups
	GroupIndex int32

	// TEMPORARY: Padding to match actual C struct size (148 bytes total)
	// We're missing 36 bytes (9 int32 fields) after GroupIndex
	// These need to be identified from Source SDK headers
	Unknown1 int32 // Offset 112
	Unknown2 int32 // Offset 116
	Unknown3 int32 // Offset 120
	Unknown4 int32 // Offset 124
	Unknown5 int32 // Offset 128
	Unknown6 int32 // Offset 132
	Unknown7 int32 // Offset 136
	Unknown8 int32 // Offset 140
	Unknown9 int32 // Offset 144
	// Total: 148 bytes
}

// Mesh represents a mesh within a model (corresponds to a single material)
// Corresponds to mstudiomesh_t in studio.h
type Mesh struct {
	// Material - ⭐ KEY FIELD! Index into the studiohdr's texture array
	Material int32

	// ModelIndex - offset back to the model (relative offset)
	ModelIndex int32

	// NumVertices - number of unique vertices in this mesh
	NumVertices int32
	// VertexOffset - offset to vertex indices (relative to model vertex data)
	VertexOffset int32

	// NumFlexes - number of flex controllers
	NumFlexes int32
	// FlexIndex - offset to flex data
	FlexIndex int32

	// MaterialType - material flags
	MaterialType int32
	// MaterialParam - material parameter
	MaterialParam int32

	// MeshID - unique mesh identifier
	MeshID int32

	// Center - mesh center point for culling
	Center mgl32.Vec3

	// MeshData - pointer to mesh vertex data (not used in file parsing)
	MeshData int32

	// NumLODVertexes - vertex counts per LOD level
	NumLODVertexes [8]int32

	_ [8]int32 // Padding/unused fields
}
