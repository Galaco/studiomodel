package vvd

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/galaco/studiomodel/internal/testutil"
)

// buildVvd builds a single-LOD vvd with two vertices, the first weighted to bone 200
func buildVvd() *testutil.Builder {
	b := &testutil.Builder{}
	// id, version, checksum, numLODs, numLODVertexes[8], numFixups, fixupTableStart, vertexDataStart, tangentDataStart
	b.U8('I', 'D', 'S', 'V').I32(4, 1234, 1, 2, 0, 0, 0, 0, 0, 0, 0, 0, 64, 64, 160)
	for _, bone := range []byte{200, 1} {
		// weights, bones, numBones, position, normal, texcoord
		b.F32(1, 0, 0).U8(bone, 0, 0, 1).F32(1, 2, 3, 0, 0, 1, 0.5, 0.5)
	}
	b.F32(1, 0, 0, 1, 1, 0, 0, 1) // tangents
	return b
}

// buildVvdWithFixups builds a two-LOD vvd with three vertices whose x position and tangent x are their file index.
// LOD 0 uses all three, in the order 2, 0, 1; LOD 1 drops vertex 0.
func buildVvdWithFixups() []byte {
	b := &testutil.Builder{}
	b.U8('I', 'D', 'S', 'V').I32(4, 1234, 2, 3, 2, 0, 0, 0, 0, 0, 0)
	b.I32(3, 64, 100, 100+3*48)      // numFixups, fixupTableStart, vertexDataStart, tangentDataStart
	b.I32(1, 2, 1, 0, 0, 1, 1, 1, 1) // fixups: lod, sourceVertexID, numVertexes
	b.Zero(100 - b.Pos())
	for v := range 3 {
		b.F32(1, 0, 0).U8(0, 0, 0, 1).F32(float32(v), 0, 0, 0, 0, 1, 0, 0)
	}
	for v := range 3 {
		b.F32(float32(v), 0, 0, 1)
	}
	return b.Bytes
}

func TestStructSizes(t *testing.T) {
	// Sizes of the matching studio.h structs
	for name, c := range map[string]struct {
		value any
		want  int
	}{
		"vertexFileHeader_t": {Header{}, 64},
		"vertexFileFixup_t":  {Fixup{}, 12},
		"mstudiovertex_t":    {Vertex{}, 48},
	} {
		if got := binary.Size(c.value); got != c.want {
			t.Errorf("%s: got %d bytes, want %d", name, got, c.want)
		}
	}
}

func TestRead(t *testing.T) {
	v, err := ReadFromStream(bytes.NewReader(buildVvd().Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Vertices) != 2 || len(v.Tangents) != 2 {
		t.Fatalf("got %d vertices, %d tangents, want 2 of each", len(v.Vertices), len(v.Tangents))
	}
	// Bone indices go up to 255, so they must not be read as signed
	if got := v.Vertices[0].BoneWeight.Bone[0]; got != 200 {
		t.Errorf("vertex 0 bone: got %d, want 200", got)
	}
}

func TestVerticesForRootLOD(t *testing.T) {
	v, err := ReadFromStream(bytes.NewReader(buildVvdWithFixups()))
	if err != nil {
		t.Fatal(err)
	}
	for rootLOD, want := range [][]float32{{2, 0, 1}, {2, 1}} {
		vertices, tangents, err := v.VerticesForRootLOD(rootLOD)
		if err != nil {
			t.Fatal(err)
		}
		if len(vertices) != len(want) || len(tangents) != len(want) {
			t.Fatalf("root LOD %d: got %d vertices, %d tangents, want %d", rootLOD, len(vertices), len(tangents), len(want))
		}
		for i := range want {
			if vertices[i].Position.X() != want[i] || tangents[i].X() != want[i] {
				t.Errorf("root LOD %d vertex %d: got vertex %v tangent %v, want file vertex %v", rootLOD, i, vertices[i].Position.X(), tangents[i].X(), want[i])
			}
		}
	}
	if _, _, err := v.VerticesForRootLOD(2); err == nil {
		t.Error("root LOD 2 of 2: expected an error")
	}
}

func TestVerticesForRootLODWithoutFixups(t *testing.T) {
	v, err := ReadFromStream(bytes.NewReader(buildVvd().Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if vertices, _, err := v.VerticesForRootLOD(0); err != nil || len(vertices) != 2 {
		t.Errorf("got %d vertices, err %v, want 2", len(vertices), err)
	}
}

func TestReadRejectsCorruptData(t *testing.T) {
	for name, corrupt := range map[string]func(b *testutil.Builder){
		"huge fixup count":      func(b *testutil.Builder) { b.SetI32(48, math.MaxInt32) },
		"negative fixup count":  func(b *testutil.Builder) { b.SetI32(48, -1) },
		"huge vertex count":     func(b *testutil.Builder) { b.SetI32(16, math.MaxInt32) },
		"negative vertex count": func(b *testutil.Builder) { b.SetI32(16, -1) },
	} {
		t.Run(name, func(t *testing.T) {
			b := buildVvd()
			corrupt(b)
			if _, err := ReadFromStream(bytes.NewReader(b.Bytes)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestReadTruncated(t *testing.T) {
	buf := buildVvd().Bytes
	for n := range len(buf) {
		if _, err := ReadFromStream(bytes.NewReader(buf[:n])); err == nil {
			t.Errorf("truncated to %d bytes: expected an error", n)
		}
	}
}

func FuzzRead(f *testing.F) {
	f.Add(buildVvd().Bytes)
	f.Add(buildVvdWithFixups())
	f.Fuzz(func(t *testing.T, data []byte) {
		v, err := ReadFromStream(bytes.NewReader(data))
		if err == nil {
			_, _, _ = v.VerticesForRootLOD(0)
		}
	})
}
