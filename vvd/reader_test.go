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

func TestStructSizes(t *testing.T) {
	// Sizes of the matching studio.h structs
	for name, c := range map[string]struct {
		value any
		want  int
	}{
		"vertexFileHeader_t": {header{}, 64},
		"vertexFileFixup_t":  {fixup{}, 12},
		"mstudiovertex_t":    {vertex{}, 48},
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

func TestReadRejectsCorruptData(t *testing.T) {
	for name, corrupt := range map[string]func(b *testutil.Builder){
		"huge fixup count":     func(b *testutil.Builder) { b.SetI32(48, math.MaxInt32) },
		"negative fixup count": func(b *testutil.Builder) { b.SetI32(48, -1) },
		"tangents before data": func(b *testutil.Builder) { b.SetI32(60, 16) },
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
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadFromStream(bytes.NewReader(data))
	})
}
