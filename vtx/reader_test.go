package vtx

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/galaco/studiomodel/internal/testutil"
)

// buildVtx builds a vtx with one body part holding a model and a blank model. The model's mesh has two strip groups,
// the second split into two strips, so the MDL v49 header layout (extended) changes where the second strip group
// and second strip are read from.
func buildVtx(extended bool) []byte {
	extra := 0
	if extended {
		extra = topologyFieldsSize
	}

	b := &testutil.Builder{}
	// header: version, vertCacheSize, maxBonesPerStrip, maxBonesPerTri, maxBonesPerVert, checkSum, numLODs,
	// materialReplacementListOffset, numBodyParts, bodyPartOffset
	b.I32(7, 24).I16(53, 9).I32(3, 1234, 1, 0, 1, 36)
	b.I32(2, 8)                // body part 0 at 36: numModels, modelOffset -> 44
	b.I32(1, 16)               // model 0 at 44: numLODs, lodOffset -> 60
	b.I32(1, 20)               // model 1 at 52 (blank): numLODs, lodOffset -> 72
	b.I32(1, 24).F32(12.5)     // model 0 LOD 0 at 60: numMeshes, meshOffset -> 84, switchPoint
	b.I32(0, 0).F32(0)         // model 1 LOD 0 at 72: no meshes
	b.I32(2, 9).U8(MeshIsEyes) // mesh 0 at 84: numStripGroups, stripGroupHeaderOffset -> 93, flags

	// strips per strip group: numIndices, indexOffset, numVerts, vertOffset
	stripGroups := [][][4]int32{
		{{3, 0, 3, 0}},
		{{3, 0, 3, 0}, {3, 3, 3, 3}},
	}
	headerPos := make([]int, len(stripGroups))
	for g, strips := range stripGroups {
		headerPos[g] = b.Pos()
		n := int32(3 * len(strips))
		// numVerts, vertOffset, numIndices, indexOffset, numStrips, stripOffset, flags
		b.I32(n, 0, n, 0, int32(len(strips)), 0).U8(StripGroupIsHWSkinned).Zero(extra)
	}
	for g, strips := range stripGroups {
		pos := headerPos[g]
		n := 3 * len(strips)
		b.SetI32(pos+4, int32(b.Pos()-pos))
		for v := range n {
			// boneWeightIndex, numBones, origMeshVertID, boneID
			b.U8(0, 1, 2, 1).I16(int16(v)).U8(0, 0, 0)
		}
		b.SetI32(pos+12, int32(b.Pos()-pos))
		for i := range n {
			b.I16(int16(i))
		}
		b.SetI32(pos+20, int32(b.Pos()-pos))
		for _, s := range strips {
			// numIndices, indexOffset, numVerts, vertOffset, numBones, flags, numBoneStateChanges, boneStateChangeOffset
			b.I32(s[0], s[1], s[2], s[3]).I16(1).U8(StripIsTriList).I32(0, 0).Zero(extra)
		}
	}

	// The one LOD's material replacements: material 2 becomes models/test/lod1
	b.SetI32(24, int32(b.Pos()))
	b.I32(1, 8)     // numReplacements, replacementOffset
	b.I16(2).I32(6) // materialID, name offset
	b.Str("models/test/lod1")

	return b.Bytes
}

func TestStructSizes(t *testing.T) {
	// Sizes of the matching optimize.h structs, which are packed
	for name, c := range map[string]struct {
		value any
		want  int
	}{
		"FileHeader_t":       {Header{}, 36},
		"BodyPartHeader_t":   {bodyPartHeader{}, 8},
		"ModelHeader_t":      {modelHeader{}, 8},
		"ModelLODHeader_t":   {modelLODHeader{}, 12},
		"MeshHeader_t":       {meshHeader{}, 9},
		"StripGroupHeader_t": {stripGroupHeader{}, 25},
		"StripHeader_t":      {Strip{}, 27},
		"Vertex_t":           {Vertex{}, 9},
	} {
		if got := binary.Size(c.value); got != c.want {
			t.Errorf("%s: got %d bytes, want %d", name, got, c.want)
		}
	}
}

func TestRead(t *testing.T) {
	for name, extended := range map[string]bool{"v48 layout": false, "v49 layout": true} {
		t.Run(name, func(t *testing.T) {
			v, err := ReadFromStream(bytes.NewReader(buildVtx(extended)))
			if err != nil {
				t.Fatal(err)
			}
			if len(v.BodyParts) != 1 || len(v.BodyParts[0].Models) != 2 {
				t.Fatalf("got %d body parts", len(v.BodyParts))
			}
			if v.Header.CheckSum != 1234 {
				t.Errorf("checksum: got %d, want 1234", v.Header.CheckSum)
			}
			want := [][]MaterialReplacement{{{MaterialID: 2, Name: "models/test/lod1"}}}
			if !reflect.DeepEqual(v.MaterialReplacements, want) {
				t.Errorf("material replacements: got %+v, want %+v", v.MaterialReplacements, want)
			}

			blank := v.BodyParts[0].Models[1]
			if len(blank.LODS) != 1 || len(blank.LODS[0].Meshes) != 0 {
				t.Errorf("blank model: got %+v", blank)
			}

			lod := v.BodyParts[0].Models[0].LODS[0]
			if lod.SwitchPoint != 12.5 {
				t.Errorf("switch point: got %v, want 12.5", lod.SwitchPoint)
			}
			meshes := lod.Meshes
			if len(meshes) != 1 || len(meshes[0].StripGroups) != 2 {
				t.Fatalf("model 0: got %+v", meshes)
			}
			if meshes[0].Flags != MeshIsEyes || meshes[0].StripGroups[0].Flags != StripGroupIsHWSkinned {
				t.Errorf("flags: got mesh %d, strip group %d", meshes[0].Flags, meshes[0].StripGroups[0].Flags)
			}
			second := meshes[0].StripGroups[1]
			if len(second.Vertexes) != 6 || second.Vertexes[5].OriginalMeshVertexID != 5 || len(second.Indices) != 6 || second.Indices[5] != 5 {
				t.Errorf("strip group 1: got %+v", second)
			}
			if len(second.Strips) != 2 || second.Strips[1].IndexOffset != 3 || second.Strips[1].VertOffset != 3 {
				t.Errorf("strip group 1 strips: got %+v", second.Strips)
			}
		})
	}
}

func TestReadWithMDLVersion(t *testing.T) {
	for _, c := range []struct {
		extended   bool
		mdlVersion int32
		wantErr    bool
	}{
		{false, 48, false},
		{true, 49, false},
		{true, 48, true},
		{false, 49, true},
	} {
		_, err := ReadFromStreamWithMDLVersion(bytes.NewReader(buildVtx(c.extended)), c.mdlVersion)
		if (err != nil) != c.wantErr {
			t.Errorf("extended=%v mdlVersion=%d: got err %v, want error %v", c.extended, c.mdlVersion, err, c.wantErr)
		}
	}
}

// Truncated files must fail cleanly rather than panic
func TestReadTruncated(t *testing.T) {
	for _, extended := range []bool{false, true} {
		buf := buildVtx(extended)
		for n := range len(buf) {
			if _, err := ReadFromStream(bytes.NewReader(buf[:n])); err == nil {
				t.Errorf("extended=%v truncated to %d bytes: expected an error", extended, n)
			}
		}
	}
}

func FuzzRead(f *testing.F) {
	f.Add(buildVtx(false))
	f.Add(buildVtx(true))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadFromStream(bytes.NewReader(data))
	})
}
