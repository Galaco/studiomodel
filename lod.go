package studiomodel

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/galaco/studiomodel/vtx"
	"github.com/galaco/studiomodel/vvd"
	"github.com/go-gl/mathgl/mgl32"
)

// LOD is a model's geometry at one level of detail, ready to draw
type LOD struct {
	// Vertices are shared by every mesh, with the vvd's fixups applied
	Vertices []vvd.Vertex
	// Tangents, one per vertex
	Tangents []mgl32.Vec4
	// Meshes of every model in every body part. To draw one bodygroup combination, pick one model per body part.
	Meshes []Mesh
}

// Mesh is one mdl mesh at one LOD: triangles drawn with one material
type Mesh struct {
	// BodyPart, Model and Mesh locate this mesh in the mdl and vtx
	BodyPart, Model, Mesh int
	// Texture is the index into the mdl's Textures and TextureNames for the chosen skin
	Texture int32
	// Material is the texture's name, or this LOD's replacement for it. Like the mdl's TextureNames, it is found
	// under one of the mdl's TextureDirs.
	Material string
	// Flags are vtx.MeshIsTeeth and vtx.MeshIsEyes
	Flags uint8
	// Indices into LOD.Vertices, three per triangle, in the file's winding order
	Indices []uint32
}

// BuildLOD assembles the model's geometry at one LOD with one skin, from its mdl, vvd and vtx
func (model *StudioModel) BuildLOD(lod int, skin int) (*LOD, error) {
	if model.Mdl == nil || model.Vvd == nil || model.Vtx == nil {
		return nil, errors.New("building a LOD needs an mdl, vvd and vtx")
	}
	if lod < 0 || lod >= int(model.Vtx.Header.NumLODs) {
		return nil, fmt.Errorf("LOD %d out of range (have %d LODs)", lod, model.Vtx.Header.NumLODs)
	}

	// Root LOD 0 keeps every vertex, so it serves every LOD with the mdl's vertex offsets as stored
	vertices, tangents, err := model.Vvd.VerticesForRootLOD(0)
	if err != nil {
		return nil, err
	}

	replacements := map[int32]string{}
	if lod < len(model.Vtx.MaterialReplacements) {
		for _, replacement := range model.Vtx.MaterialReplacements[lod] {
			replacements[int32(replacement.MaterialID)] = replacement.Name
		}
	}

	out := &LOD{
		Vertices: vertices,
		Tangents: tangents,
	}

	if len(model.Vtx.BodyParts) != len(model.Mdl.BodyParts) {
		return nil, fmt.Errorf("vtx has %d body parts, mdl has %d", len(model.Vtx.BodyParts), len(model.Mdl.BodyParts))
	}
	vertexSize := int64(binary.Size(vvd.Vertex{}))
	for i, bodyPart := range model.Vtx.BodyParts {
		mdlBodyPart := &model.Mdl.BodyParts[i]
		if len(bodyPart.Models) != len(mdlBodyPart.Models) {
			return nil, fmt.Errorf("body part %d: vtx has %d models, mdl has %d", i, len(bodyPart.Models), len(mdlBodyPart.Models))
		}

		for j, vtxModel := range bodyPart.Models {
			mdlModel := &mdlBodyPart.Models[j]
			if lod >= len(vtxModel.LODS) {
				return nil, fmt.Errorf("body part %d model %d: LOD %d out of range (have %d LODs)", i, j, lod, len(vtxModel.LODS))
			}
			vtxMeshes := vtxModel.LODS[lod].Meshes
			if len(vtxMeshes) != len(mdlModel.Meshes) {
				return nil, fmt.Errorf("body part %d model %d: vtx has %d meshes, mdl has %d", i, j, len(vtxMeshes), len(mdlModel.Meshes))
			}

			// The model's VertexIndex is a byte offset into the vvd's vertices
			modelStart := int64(mdlModel.Header.VertexIndex) / vertexSize
			for k, vtxMesh := range vtxMeshes {
				mdlMesh := &mdlModel.Meshes[k]
				texture, err := model.Mdl.TextureIndex(skin, mdlMesh.Material)
				if err != nil {
					return nil, fmt.Errorf("body part %d model %d mesh %d: %w", i, j, k, err)
				}
				material, ok := replacements[texture]
				if !ok {
					if texture < 0 || int(texture) >= len(model.Mdl.TextureNames) {
						return nil, fmt.Errorf("body part %d model %d mesh %d: texture %d out of range (have %d)", i, j, k, texture, len(model.Mdl.TextureNames))
					}
					material = model.Mdl.TextureNames[texture]
				}

				indices, err := triangles(vtxMesh, modelStart+int64(mdlMesh.VertexOffset), len(vertices))
				if err != nil {
					return nil, fmt.Errorf("body part %d model %d mesh %d: %w", i, j, k, err)
				}

				out.Meshes = append(out.Meshes, Mesh{
					BodyPart: i,
					Model:    j,
					Mesh:     k,
					Texture:  texture,
					Material: material,
					Flags:    vtxMesh.Flags,
					Indices:  indices,
				})
			}
		}
	}

	return out, nil
}

// triangles returns a vtx mesh's triangles as indices into the vvd's vertices, given the index of the mesh's first
// vertex there
func triangles(mesh vtx.Mesh, meshStart int64, numVertices int) ([]uint32, error) {
	var out []uint32
	for g, group := range mesh.StripGroups {
		// A strip's indices point into its strip group's vertices, which point into the mesh's vertices
		vertex := func(index int) (uint32, error) {
			v := int(group.Indices[index])
			if v >= len(group.Vertexes) {
				return 0, fmt.Errorf("strip group %d index %d points to vertex %d of %d", g, index, v, len(group.Vertexes))
			}
			global := meshStart + int64(group.Vertexes[v].OriginalMeshVertexID)
			if global < 0 || global >= int64(numVertices) {
				return 0, fmt.Errorf("strip group %d vertex %d is vvd vertex %d of %d", g, v, global, numVertices)
			}
			return uint32(global), nil
		}

		for s, strip := range group.Strips {
			start, end := int(strip.IndexOffset), int(strip.IndexOffset)+int(strip.NumIndices)
			if start < 0 || end < start || end > len(group.Indices) {
				return nil, fmt.Errorf("strip group %d strip %d indices %d to %d out of range (have %d)", g, s, start, end, len(group.Indices))
			}

			step := 3
			if strip.Flags&vtx.StripIsTriStrip != 0 {
				step = 1
			}
			for i := start; i+2 < end; i += step {
				var triangle [3]uint32
				for c := range triangle {
					v, err := vertex(i + c)
					if err != nil {
						return nil, err
					}
					triangle[c] = v
				}
				if step == 1 {
					// Every other triangle of a strip is wound the other way, and repeated vertices join strips
					if (i-start)%2 == 1 {
						triangle[1], triangle[2] = triangle[2], triangle[1]
					}
					if triangle[0] == triangle[1] || triangle[1] == triangle[2] || triangle[0] == triangle[2] {
						continue
					}
				}
				out = append(out, triangle[:]...)
			}
		}
	}
	return out, nil
}
