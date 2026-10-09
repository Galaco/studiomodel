package studiomodel

import (
	"slices"
	"testing"

	"github.com/galaco/studiomodel/mdl"
	"github.com/galaco/studiomodel/vtx"
	"github.com/galaco/studiomodel/vvd"
	"github.com/go-gl/mathgl/mgl32"
)

// testModel builds a two-LOD model with one body part of two models. Model 0 is a single triangle with one texture
// per skin; model 1 is a triangle strip of two triangles starting at vvd vertex 3. Its vvd stores vertices in reverse,
// with fixups to put them back in order, and its LOD 1 replaces texture 0.
func testModel() *StudioModel {
	vertices := make([]vvd.Vertex, 7)
	for i := range vertices {
		vertices[i].Position[0] = float32(6 - i) // stored in reverse
	}
	fixups := make([]vvd.Fixup, 7)
	for i := range fixups {
		fixups[i] = vvd.Fixup{Lod: 1, SourceVertexID: int32(6 - i), NumVertexes: 1}
	}

	triangle := vtx.Mesh{StripGroups: []vtx.StripGroup{{
		Vertexes: []vtx.Vertex{{OriginalMeshVertexID: 0}, {OriginalMeshVertexID: 1}, {OriginalMeshVertexID: 2}},
		Indices:  []uint16{0, 1, 2},
		Strips:   []vtx.Strip{{NumIndices: 3, NumVerts: 3, Flags: vtx.StripIsTriList}},
	}}}
	strip := vtx.Mesh{StripGroups: []vtx.StripGroup{{
		Vertexes: []vtx.Vertex{{OriginalMeshVertexID: 0}, {OriginalMeshVertexID: 1}, {OriginalMeshVertexID: 2}, {OriginalMeshVertexID: 3}},
		Indices:  []uint16{0, 1, 2, 3},
		Strips:   []vtx.Strip{{NumIndices: 4, NumVerts: 4, Flags: vtx.StripIsTriStrip}},
	}}, Flags: vtx.MeshIsEyes}

	return &StudioModel{
		Mdl: &mdl.Mdl{
			TextureNames: []string{"skin0", "skin1"},
			SkinFamilies: [][]int16{{0}, {1}},
			BodyParts: []mdl.BodyPartData{{Models: []mdl.ModelData{
				{Meshes: []mdl.Mesh{{Material: 0}}},
				{Header: mdl.Model{VertexIndex: 3 * 48}, Meshes: []mdl.Mesh{{Material: 0}}},
			}}},
		},
		Vvd: &vvd.Vvd{
			Header:   vvd.Header{NumLODs: 2, NumLODVertexes: [vvd.MaxNumLods]int32{7, 7}},
			Fixups:   fixups,
			Vertices: vertices,
			Tangents: make([]mgl32.Vec4, 7),
		},
		Vtx: &vtx.Vtx{
			Header: vtx.Header{NumLODs: 2},
			BodyParts: []vtx.BodyPart{{Models: []vtx.Model{
				{LODS: []vtx.ModelLOD{{Meshes: []vtx.Mesh{triangle}}, {Meshes: []vtx.Mesh{triangle}}}},
				{LODS: []vtx.ModelLOD{{Meshes: []vtx.Mesh{strip}}, {Meshes: []vtx.Mesh{strip}}}},
			}}},
			MaterialReplacements: [][]vtx.MaterialReplacement{nil, {{MaterialID: 0, Name: "lod1"}}},
		},
	}
}

func TestBuildLOD(t *testing.T) {
	for _, c := range []struct {
		lod, skin int
		materials []string
	}{
		{0, 0, []string{"skin0", "skin0"}},
		{0, 1, []string{"skin1", "skin1"}},
		{1, 0, []string{"lod1", "lod1"}},
		{1, 1, []string{"skin1", "skin1"}},
	} {
		lod, err := testModel().BuildLOD(c.lod, c.skin)
		if err != nil {
			t.Fatal(err)
		}

		// Fixups put the vertices back in order
		for i, v := range lod.Vertices {
			if v.Position.X() != float32(i) {
				t.Fatalf("LOD %d vertex %d: got file vertex %v", c.lod, i, v.Position.X())
			}
		}

		if len(lod.Meshes) != 2 {
			t.Fatalf("LOD %d: got %d meshes, want 2", c.lod, len(lod.Meshes))
		}
		triangle, strip := lod.Meshes[0], lod.Meshes[1]
		if !slices.Equal(triangle.Indices, []uint32{0, 1, 2}) {
			t.Errorf("LOD %d triangle: got indices %v", c.lod, triangle.Indices)
		}
		// The strip's second triangle is wound the other way, and the model starts at vertex 3
		if !slices.Equal(strip.Indices, []uint32{3, 4, 5, 4, 6, 5}) || strip.Model != 1 || strip.Flags != vtx.MeshIsEyes {
			t.Errorf("LOD %d strip: got %+v", c.lod, strip)
		}
		if got := []string{triangle.Material, strip.Material}; !slices.Equal(got, c.materials) {
			t.Errorf("LOD %d skin %d: got materials %q, want %q", c.lod, c.skin, got, c.materials)
		}
	}
}

func TestBuildLODRejectsBadInput(t *testing.T) {
	for name, c := range map[string]struct {
		corrupt   func(m *StudioModel)
		lod, skin int
	}{
		"LOD out of range":  {func(m *StudioModel) {}, 2, 0},
		"skin out of range": {func(m *StudioModel) {}, 0, 2},
		"missing vvd":       {func(m *StudioModel) { m.Vvd = nil }, 0, 0},
		"index past vertices": {func(m *StudioModel) {
			m.Vtx.BodyParts[0].Models[1].LODS[0].Meshes[0].StripGroups[0].Vertexes[3].OriginalMeshVertexID = 4
		}, 0, 0},
		"mismatched meshes": {func(m *StudioModel) { m.Mdl.BodyParts[0].Models[0].Meshes = nil }, 0, 0},
	} {
		t.Run(name, func(t *testing.T) {
			m := testModel()
			c.corrupt(m)
			if _, err := m.BuildLOD(c.lod, c.skin); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	m := testModel()
	m.Mdl.Header.Checksum = 5
	m.Vvd.Header.Checksum = 5
	m.Vtx.Header.CheckSum = 5
	if err := m.Validate(); err != nil {
		t.Error(err)
	}
	m.Vtx.Header.CheckSum = 6
	if err := m.Validate(); err == nil {
		t.Error("mismatched vtx checksum: expected an error")
	}
}
