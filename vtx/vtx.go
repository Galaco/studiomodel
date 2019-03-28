package vtx

// Vtx
type Vtx struct {
	// BodyParts
	BodyParts []BodyPart
}

// BodyPart
type BodyPart struct {
	// Models
	Models []Model
}

// Model
type Model struct {
	// LODS
	LODS []ModelLOD
}

// ModelLOD
type ModelLOD struct {
	// Meshes
	Meshes []Mesh
}

// Mesh
type Mesh struct {
	// StripGroups
	StripGroups []StripGroup
}

// StripGroup
type StripGroup struct {
	// Indices
	Indices []uint16
	// Vertexes
	Vertexes []Vertex
	// Strips
	Strips []Strip
}

// header
type header struct {
	// Version
	Version int32
	// VertCacheSize
	VertCacheSize int32

	// MaxBonesPerStrip
	MaxBonesPerStrip uint16
	// MaxBonesPerTriangle
	MaxBonesPerTriangle uint16
	// MaxBonesPerVert
	MaxBonesPerVert int32

	// CheckSum
	CheckSum int32
	// NumLODs
	NumLODs int32

	// MaterialReplacementListOffset
	MaterialReplacementListOffset int32

	// NumBodyParts
	NumBodyParts int32
	// BodyPartOffset
	BodyPartOffset int32
}

// bodyPartHeader
type bodyPartHeader struct {
	// NumModels
	NumModels int32
	// ModelOffset
	ModelOffset int32
}

// modelHeader
type modelHeader struct {
	// NumLODs
	NumLODs int32
	// LODOffset
	LODOffset int32
}

// modelLODHeader
type modelLODHeader struct {
	// NumMeshes
	NumMeshes int32
	// MeshOffset
	MeshOffset int32
	// SwitchPoint
	SwitchPoint float32
}

// meshHeader
type meshHeader struct {
	// NumStripGroups
	NumStripGroups int32
	// StripGroupHeaderOffset
	StripGroupHeaderOffset int32

	// Flags
	Flags uint8
}

const (
	// StripGroupIsFlexed
	StripGroupIsFlexed = 0x01
	// StripGroupIsHWSkinned
	StripGroupIsHWSkinned = 0x02
	// StripGroupIsDeltaFlexed
	StripGroupIsDeltaFlexed = 0x04
	// StripGroupSuppressHWMorph
	StripGroupSuppressHWMorph = 0x08
)

// stripGroupHeader
type stripGroupHeader struct {
	// NumVerts
	NumVerts int32
	// VertOffset
	VertOffset int32

	// NumIndices
	NumIndices int32
	// IndexOffset
	IndexOffset int32

	// NumStrips
	NumStrips int32
	// StripOffset
	StripOffset int32

	// Flags
	Flags uint8
	//_     [3]byte
}

// Strip
type Strip struct {
	// NumIndices
	NumIndices int32
	// IndexOffset
	IndexOffset int32

	// NumVerts
	NumVerts int32
	// VertOffset
	VertOffset int32

	// NumBones
	NumBones int16

	// Flags
	Flags uint8
	//_     byte

	//NumBoneStateChanges
	NumBoneStateChanges int32
	//BoneStateChangeOffset
	BoneStateChangeOffset int32
}

// Vertex
type Vertex struct {
	// BoneWeightIndex
	BoneWeightIndex [3]uint8
	// NumBones
	NumBones uint8

	// OriginalMeshVertexID
	OriginalMeshVertexID uint16

	// BoneID
	BoneID [3]int8
}
