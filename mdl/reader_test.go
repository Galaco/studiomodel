package mdl

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/galaco/studiomodel/internal"
	"github.com/galaco/studiomodel/internal/testutil"
	"github.com/go-gl/mathgl/mgl32"
)

// Offsets of the test file built by buildMdl. Field offsets inside each struct follow studio.h.
const (
	bonesAt     = 408
	boneSize    = 216
	namesAt     = 840
	bodyPartAt  = 852
	modelsAt    = 868
	modelSize   = 148
	meshAt      = 1164
	textureAt   = 1280
	texNameAt   = 1344
	cdOffsetsAt = 1356
	cdPathAt    = 1360
	hdr2At      = 1376
	fileSize    = 1632
)

// buildMdl builds a v48 mdl with two bones, one texture, and a body part holding a model and a blank model
func buildMdl() *testutil.Builder {
	b := (&testutil.Builder{}).Zero(fileSize)
	copy(b.Bytes, "IDST")
	b.SetI32(4, 48)
	b.SetI32(8, 1234)
	copy(b.Bytes[12:], "test.mdl")
	b.SetI32(76, fileSize)
	b.SetI32(156, 2)
	b.SetI32(160, bonesAt)
	b.SetI32(204, 1)
	b.SetI32(208, textureAt)
	b.SetI32(212, 1)
	b.SetI32(216, cdOffsetsAt)
	b.SetI32(232, 1)
	b.SetI32(236, bodyPartAt)
	b.SetF32(392, 0.5) // flVertAnimFixedPointScale
	b.SetI32(400, hdr2At)

	copy(b.Bytes[namesAt:], "root\x00child\x00")
	b.SetI32(bonesAt, namesAt-bonesAt)
	b.SetI32(bonesAt+4, -1)
	for i, v := range []float32{0.1, 0.2, 0.3, 0.9} { // quat, x y z w
		b.SetF32(bonesAt+44+4*i, v)
	}
	for i, v := range []float32{1, 0, 0, 10, 0, 1, 0, 20, 0, 0, 1, 30} { // poseToBone, row by row
		b.SetF32(bonesAt+96+4*i, v)
	}
	b.SetI32(bonesAt+172, 7)    // physicsbone
	b.SetI32(bonesAt+176, 1234) // surfacepropidx
	b.SetI32(bonesAt+180, 1)    // contents
	const child = bonesAt + boneSize
	b.SetI32(child, namesAt+5-child)
	b.SetI32(child+4, 0)

	b.SetI32(bodyPartAt+4, 2)
	b.SetI32(bodyPartAt+12, modelsAt-bodyPartAt)
	copy(b.Bytes[modelsAt:], "body")
	b.SetI32(modelsAt+72, 1)
	b.SetI32(modelsAt+76, meshAt-modelsAt)
	copy(b.Bytes[modelsAt+modelSize:], "blank")
	b.SetI32(meshAt, 3)

	b.SetI32(textureAt, texNameAt-textureAt)
	copy(b.Bytes[texNameAt:], "metal/floor\x00")
	b.SetI32(cdOffsetsAt, cdPathAt)
	copy(b.Bytes[cdPathAt:], "models/test/\x00")

	return b
}

func TestStructSizes(t *testing.T) {
	// Sizes of the matching studio.h structs
	for name, c := range map[string]struct {
		value any
		want  int
	}{
		"studiohdr_t":             {Studiohdr{}, 408},
		"mstudiobone_t":           {Bone{}, 216},
		"mstudiobonecontroller_t": {BoneController{}, 56},
		"mstudiohitboxset_t":      {HitboxSet{}, 12},
		"mstudioanimdesc_t":       {AnimDesc{}, 100},
		"mstudioseqdesc_t":        {SequenceDesc{}, 212},
		"mstudiotexture_t":        {Texture{}, 64},
		"mstudiobodyparts_t":      {BodyPart{}, 16},
		"mstudiomodel_t":          {Model{}, 148},
		"mstudiomesh_t":           {Mesh{}, 116},
	} {
		if got := binary.Size(c.value); got != c.want {
			t.Errorf("%s: got %d bytes, want %d", name, got, c.want)
		}
	}
}

func TestRead(t *testing.T) {
	buf := buildMdl().Bytes
	m, err := ReadFromStream(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

	for i, want := range []string{"root", "child"} {
		offset := int64(m.Header.BoneOffset) + int64(i*boneSize) + int64(m.Bones[i].NameIndex)
		if got, _ := internal.CString(buf, offset, 64, "bone name"); got != want {
			t.Errorf("bone %d name: got %q, want %q", i, got, want)
		}
	}
	if m.Bones[1].Parent != 0 {
		t.Errorf("bone 1 parent: got %d, want 0", m.Bones[1].Parent)
	}

	bone := m.Bones[0]
	if bone.PhysicsBone != 7 || bone.SurfacePropIndex != 1234 || bone.Contents != 1 {
		t.Errorf("bone 0: got physicsbone=%d surfaceprop=%d contents=%d, want 7, 1234, 1", bone.PhysicsBone, bone.SurfacePropIndex, bone.Contents)
	}
	if got, want := bone.Quaternion.Quat(), (mgl32.Quat{W: 0.9, V: mgl32.Vec3{0.1, 0.2, 0.3}}); got != want {
		t.Errorf("bone 0 quaternion: got %v, want %v", got, want)
	}
	if got, want := bone.PoseToBone.Mat3x4().Col(3), (mgl32.Vec3{10, 20, 30}); got != want {
		t.Errorf("bone 0 poseToBone translation: got %v, want %v", got, want)
	}

	if m.Header.StudioHDR2Index != hdr2At {
		t.Errorf("StudioHDR2Index: got %d, want %d", m.Header.StudioHDR2Index, hdr2At)
	}
	if m.Header.VertAnimFixedPointScale != 0.5 {
		t.Errorf("VertAnimFixedPointScale: got %v, want 0.5", m.Header.VertAnimFixedPointScale)
	}

	if !slices.Equal(m.TextureNames, []string{"metal/floor"}) || !slices.Equal(m.TextureDirs, []string{"models/test/"}) {
		t.Errorf("textures: got names %q dirs %q", m.TextureNames, m.TextureDirs)
	}

	if len(m.BodyParts) != 1 || len(m.BodyParts[0].Models) != 2 {
		t.Fatalf("body parts: got %+v", m.BodyParts)
	}
	if meshes := m.BodyParts[0].Models[0].Meshes; len(meshes) != 1 || meshes[0].Material != 3 {
		t.Errorf("model 0 meshes: got %+v", meshes)
	}
	if meshes := m.BodyParts[0].Models[1].Meshes; len(meshes) != 0 {
		t.Errorf("blank model meshes: got %+v", meshes)
	}
}

func TestReadToleratesTrailingBytes(t *testing.T) {
	b := buildMdl().Zero(16)
	if _, err := ReadFromStream(bytes.NewReader(b.Bytes)); err != nil {
		t.Error(err)
	}
}

func TestReadRejectsCorruptData(t *testing.T) {
	for name, corrupt := range map[string]func(b *testutil.Builder){
		"negative model count": func(b *testutil.Builder) { b.SetI32(bodyPartAt+4, -1) },
		"negative mesh count":  func(b *testutil.Builder) { b.SetI32(modelsAt+72, -1) },
		"huge bone count":      func(b *testutil.Builder) { b.SetI32(156, math.MaxInt32) },
		"huge mesh count":      func(b *testutil.Builder) { b.SetI32(modelsAt+72, math.MaxInt32) },
		"bones past end":       func(b *testutil.Builder) { b.SetI32(160, fileSize-boneSize) },
		"model offset wraps":   func(b *testutil.Builder) { b.SetI32(bodyPartAt+12, math.MaxInt32) },
		"length past end":      func(b *testutil.Builder) { b.SetI32(76, fileSize+1) },
	} {
		t.Run(name, func(t *testing.T) {
			b := buildMdl()
			corrupt(b)
			if _, err := ReadFromStream(bytes.NewReader(b.Bytes)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestReadTruncated(t *testing.T) {
	buf := buildMdl().Bytes
	for n := range len(buf) {
		if _, err := ReadFromStream(bytes.NewReader(buf[:n])); err == nil {
			t.Errorf("truncated to %d bytes: expected an error", n)
		}
	}
}

func FuzzRead(f *testing.F) {
	f.Add(buildMdl().Bytes)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadFromStream(bytes.NewReader(data))
	})
}
