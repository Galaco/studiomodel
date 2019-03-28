package vvd

import "github.com/go-gl/mathgl/mgl32"

const (
	// MaxNumLods is the maximum number of LODs a model can have
	MaxNumLods = 8
	// MaxNumBonesPerVertex is the maximum number of bones that a particular vertex can be weighted against
	MaxNumBonesPerVertex = 3
)

// Vvd
type Vvd struct {
	// Header
	Header header
	// Fixups
	Fixups []fixup
	// Vertices
	Vertices []vertex
	// Tangents
	Tangents []mgl32.Vec4
}

// header
type header struct {
	// Id
	Id int32
	// Version
	Version int32
	// Checksum
	Checksum int32
	// NumLODs
	NumLODs int32
	// NumLODVertexes
	NumLODVertexes [MaxNumLods]int32
	// NumFixups
	NumFixups int32
	// FixupTableStart
	FixupTableStart int32
	// VertexDataStart
	VertexDataStart int32
	// TangentDataStart
	TangentDataStart int32
}

// fixup
type fixup struct {
	// Lod
	Lod int32
	// SourceVertexID
	SourceVertexID int32
	// NumVertexes
	NumVertexes int32
}

// vertex
type vertex struct {
	// BoneWeight
	BoneWeight boneWeight
	// Position
	Position mgl32.Vec3
	// Normal
	Normal mgl32.Vec3
	// UVs
	UVs mgl32.Vec2
}

// boneWeight
type boneWeight struct {
	// Weight
	Weight [MaxNumBonesPerVertex]float32
	// Bone
	Bone [MaxNumBonesPerVertex]int8
	// NumBones
	NumBones int8
}
