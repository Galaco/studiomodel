package vvd

import (
	"fmt"

	"github.com/go-gl/mathgl/mgl32"
)

const (
	// MaxNumLods is the maximum number of LODs a model can have
	MaxNumLods = 8
	// MaxNumBonesPerVertex is the maximum number of bones that a particular vertex can be weighted against
	MaxNumBonesPerVertex = 3
)

// Vvd
type Vvd struct {
	// Header
	Header Header
	// Fixups
	Fixups []Fixup
	// Vertices are stored in file order. When there are fixups this is not the order mdl and vtx indices expect;
	// use VerticesForRootLOD.
	Vertices []Vertex
	// Tangents
	Tangents []mgl32.Vec4
}

// VerticesForRootLOD returns the vertices and tangents in the order the engine uses when rootLOD is the most
// detailed LOD it loads, with fixups applied.
//
// Root LOD 0 serves every LOD: a vertex of any vtx LOD is at mdl model VertexIndex/48 + mesh VertexOffset + the vtx
// vertex's OriginalMeshVertexID. A higher root LOD drops vertices that only more detailed LODs use, so mesh vertex
// offsets must then be recomputed from Mesh.NumLODVertexes as the engine does.
func (vvd *Vvd) VerticesForRootLOD(rootLOD int) ([]Vertex, []mgl32.Vec4, error) {
	if rootLOD < 0 || rootLOD >= int(vvd.Header.NumLODs) {
		return nil, nil, fmt.Errorf("root LOD %d out of range (have %d LODs)", rootLOD, vvd.Header.NumLODs)
	}

	// Without fixups, every LOD's vertices are a prefix of the file's
	if len(vvd.Fixups) == 0 {
		count := max(0, min(int(vvd.Header.NumLODVertexes[rootLOD]), len(vvd.Vertices), len(vvd.Tangents)))
		return vvd.Vertices[:count], vvd.Tangents[:count], nil
	}

	var vertices []Vertex
	var tangents []mgl32.Vec4
	for i, fixup := range vvd.Fixups {
		if int(fixup.Lod) < rootLOD {
			continue
		}
		start, end := int64(fixup.SourceVertexID), int64(fixup.SourceVertexID)+int64(fixup.NumVertexes)
		if start < 0 || end < start || end > int64(len(vvd.Vertices)) || end > int64(len(vvd.Tangents)) {
			return nil, nil, fmt.Errorf("fixup %d (vertices %d to %d) out of range (have %d vertices)", i, start, end, len(vvd.Vertices))
		}
		vertices = append(vertices, vvd.Vertices[start:end]...)
		tangents = append(tangents, vvd.Tangents[start:end]...)
	}
	return vertices, tangents, nil
}

// Header
type Header struct {
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

// Fixup
type Fixup struct {
	// Lod
	Lod int32
	// SourceVertexID
	SourceVertexID int32
	// NumVertexes
	NumVertexes int32
}

// Vertex
type Vertex struct {
	// BoneWeight
	BoneWeight BoneWeight
	// Position
	Position mgl32.Vec3
	// Normal
	Normal mgl32.Vec3
	// UVs
	UVs mgl32.Vec2
}

// BoneWeight
type BoneWeight struct {
	// Weight
	Weight [MaxNumBonesPerVertex]float32
	// Bone
	Bone [MaxNumBonesPerVertex]uint8
	// NumBones
	NumBones int8
}
