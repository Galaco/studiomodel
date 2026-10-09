package mdl

import "github.com/go-gl/mathgl/mgl32"

// Quaternion is stored on disk as x, y, z, w (mgl32.Quat stores w first, so it cannot be read into directly).
type Quaternion struct {
	X, Y, Z, W float32
}

// Quat returns q as an mgl32.Quat
func (q Quaternion) Quat() mgl32.Quat {
	return mgl32.Quat{W: q.W, V: mgl32.Vec3{q.X, q.Y, q.Z}}
}

// Matrix3x4 is stored on disk row by row (Valve's matrix3x4_t). mgl32.Mat3x4 is column-major, so it cannot be read
// into directly.
type Matrix3x4 [3][4]float32

// Mat3x4 returns m as an mgl32.Mat3x4
func (m Matrix3x4) Mat3x4() mgl32.Mat3x4 {
	return mgl32.Mat3x4FromRows(mgl32.Vec4(m[0]), mgl32.Vec4(m[1]), mgl32.Vec4(m[2]))
}

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
	Quaternion Quaternion
	// Rotation
	Rotation mgl32.Vec3

	// PosScale
	PosScale mgl32.Vec3
	// RotScale
	RotScale mgl32.Vec3

	// PoseToBone
	PoseToBone Matrix3x4
	// Alignment
	Alignment Quaternion

	// Flags
	Flags int32
	// ProcType
	ProcType int32
	// ProcIndex
	ProcIndex int32
	// PhysicsBone
	PhysicsBone int32
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

	// NumVertices - unique vertices in this model
	NumVertices int32
	// VertexIndex - byte offset of this model's first vertex in the vvd's vertex data (48 bytes per vertex)
	VertexIndex int32
	// TangentsIndex - byte offset of this model's first tangent in the vvd's tangent data (16 bytes per tangent)
	TangentsIndex int32

	// NumAttachments
	NumAttachments int32
	// AttachmentIndex
	AttachmentIndex int32

	// NumEyeballs
	NumEyeballs int32
	// EyeballIndex - byte offset from start of this struct to first eyeball
	EyeballIndex int32

	// vertex and tangent data pointers, set by the engine at load time
	_ [2]int32
	_ [8]int32
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
	// VertexOffset - index of this mesh's first vertex within its model's vertices
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
