package mdl

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
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
	texNamesAt  = 1408
	cdOffsetsAt = 1432
	cdPathAt    = 1436
	skinsAt     = 1452
	metalAt     = 1456
	bodyNameAt  = 1462
	keyValuesAt = 1467
	hdr2At      = 1484
	baseSize    = 1740 // rigging sections follow; see addRigging

	testKeyValues = "mdlkeyvalue { }"
)

// buildMdl builds a v48 mdl with two bones, two textures, two skins, and a body part holding a model and a blank
// model, followed by the sections addRigging describes
func buildMdl() *testutil.Builder {
	b := (&testutil.Builder{}).Zero(baseSize)
	copy(b.Bytes, "IDST")
	b.SetI32(4, 48)
	b.SetI32(8, 1234)
	copy(b.Bytes[12:], "test.mdl")
	b.SetI32(156, 2)
	b.SetI32(160, bonesAt)
	b.SetI32(204, 2)
	b.SetI32(208, textureAt)
	b.SetI32(212, 1)
	b.SetI32(216, cdOffsetsAt)
	b.SetI32(220, 1) // numskinref
	b.SetI32(224, 2) // numskinfamilies
	b.SetI32(228, skinsAt)
	b.SetI32(232, 1)
	b.SetI32(236, bodyPartAt)
	b.SetI32(308, metalAt) // surfacepropindex
	b.SetI32(312, keyValuesAt)
	b.SetI32(316, int32(len(testKeyValues)+1))
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
	b.SetI32(bonesAt+172, 7)               // physicsbone
	b.SetI32(bonesAt+176, metalAt-bonesAt) // surfacepropidx
	b.SetI32(bonesAt+180, 1)               // contents
	const child = bonesAt + boneSize
	b.SetI32(child, namesAt+5-child)
	b.SetI32(child+4, 0)
	b.SetI32(child+176, metalAt-child)

	b.SetI32(bodyPartAt, bodyNameAt-bodyPartAt)
	b.SetI32(bodyPartAt+4, 2)
	b.SetI32(bodyPartAt+12, modelsAt-bodyPartAt)
	copy(b.Bytes[modelsAt:], "body")
	b.SetI32(modelsAt+72, 1)
	b.SetI32(modelsAt+76, meshAt-modelsAt)
	copy(b.Bytes[modelsAt+modelSize:], "blank")
	b.SetI32(meshAt, 0) // material: skin slot 0

	b.SetI32(textureAt, texNamesAt-textureAt)
	b.SetI32(textureAt+64, texNamesAt+12-(textureAt+64))
	copy(b.Bytes[texNamesAt:], "metal/floor\x00metal/rust\x00")
	b.SetI32(cdOffsetsAt, cdPathAt)
	copy(b.Bytes[cdPathAt:], "models/test/\x00")

	// skin 0 uses texture 0 for slot 0, skin 1 uses texture 1
	b.Bytes[skinsAt+2] = 1

	copy(b.Bytes[metalAt:], "metal\x00body\x00"+testKeyValues+"\x00")

	addRigging(b)
	b.SetI32(76, int32(len(b.Bytes))) // length
	return b
}

const testLongName = "models/a/model/whose/full/name/is/longer/than/the/sixty/four/characters/a/header/holds.mdl"

// addRigging appends a hitbox set of two hitboxes, an attachment, a pose parameter, an IK chain and lock, an include
// model, a quaternion interpolation rule for bone 0, a jiggle bone rule for bone 1, and a source bone transform with
// the secondary header pointing at it
func addRigging(b *testutil.Builder) {
	reserve := func(n int) int {
		at := b.Pos()
		b.Zero(n)
		return at
	}
	hitboxSet, hitboxes, attachment, poseParam := reserve(12), reserve(2*68), reserve(92), reserve(20)
	ikChain, ikLock, include := reserve(16+2*28), reserve(32), reserve(8)
	quatInterp, jiggle, srcTransform := reserve(12+48), reserve(120+20), reserve(100)

	str := func(s string) int {
		at := b.Pos()
		b.Str(s)
		return at
	}
	defaultAt, headAt, muzzleAt, aimAt, legAt := str("default"), str("head"), str("muzzle"), str("aim_yaw"), str("leg")
	animsAt, rootAt, longNameAt := str("models/anims.mdl"), str("root"), str(testLongName)

	setVec := func(at int, values ...float32) {
		for i, v := range values {
			b.SetF32(at+4*i, v)
		}
	}

	b.SetI32(172, 1)
	b.SetI32(176, int32(hitboxSet))
	b.SetI32(hitboxSet, int32(defaultAt-hitboxSet))
	b.SetI32(hitboxSet+4, 2)
	b.SetI32(hitboxSet+8, int32(hitboxes-hitboxSet))
	// hitbox 0: bone, group, bbmin, bbmax, name, CS:GO angles and capsule radius
	b.SetI32(hitboxes, 1)
	b.SetI32(hitboxes+4, 3)
	setVec(hitboxes+8, -1, -2, -3, 1, 2, 3)
	b.SetI32(hitboxes+32, int32(headAt-hitboxes))
	setVec(hitboxes+36, 0, 90, 0, 2.5)
	// hitbox 1's name offset is from the start of the file, as some v44 files have it
	b.SetI32(hitboxes+68+32, int32(headAt))

	b.SetI32(240, 1)
	b.SetI32(244, int32(attachment))
	b.SetI32(attachment, int32(muzzleAt-attachment))
	b.SetI32(attachment+8, 1)
	setVec(attachment+12, 1, 0, 0, 1, 0, 1, 0, 2, 0, 0, 1, 3)

	b.SetI32(300, 1)
	b.SetI32(304, int32(poseParam))
	b.SetI32(poseParam, int32(aimAt-poseParam))
	setVec(poseParam+8, -45, 45, 0)

	b.SetI32(284, 1)
	b.SetI32(288, int32(ikChain))
	b.SetI32(ikChain, int32(legAt-ikChain))
	b.SetI32(ikChain+8, 2)
	b.SetI32(ikChain+12, 16)
	b.SetI32(ikChain+16, 0)
	setVec(ikChain+20, 0, 1, 0)
	b.SetI32(ikChain+44, 1)
	setVec(ikChain+48, 0, 0, 1)

	b.SetI32(320, 1)
	b.SetI32(324, int32(ikLock))
	setVec(ikLock+4, 1, 0.5)

	b.SetI32(336, 1)
	b.SetI32(340, int32(include))
	b.SetI32(include+4, int32(animsAt-include))

	// bone 0: quaternion interpolation controlled by bone 1, with one trigger
	b.SetI32(bonesAt+164, 2)
	b.SetI32(bonesAt+168, int32(quatInterp-bonesAt))
	b.SetI32(quatInterp, 1)
	b.SetI32(quatInterp+4, 1)
	b.SetI32(quatInterp+8, 12)
	setVec(quatInterp+12, 2, 0, 0, 0, 1, 1, 0, 0, 0, 0, 1, 0)

	// bone 1: a flexible jiggle bone with boing
	const child = bonesAt + boneSize
	b.SetI32(child+164, 5)
	b.SetI32(child+168, int32(jiggle-child))
	b.SetI32(jiggle, JiggleIsFlexible|JiggleIsBoing)
	setVec(jiggle+4, 10, 2)
	setVec(jiggle+120, 100)

	// The source bone transform's offset is from the main header, its name's from the transform
	b.SetI32(hdr2At, 1)
	b.SetI32(hdr2At+4, int32(srcTransform))
	b.SetI32(hdr2At+20, int32(longNameAt-hdr2At))
	b.SetI32(srcTransform, int32(rootAt-srcTransform))
	setVec(srcTransform+4, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 5)
	setVec(srcTransform+52, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0)
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
	if bone.PhysicsBone != 7 || bone.Contents != 1 {
		t.Errorf("bone 0: got physicsbone=%d contents=%d, want 7, 1", bone.PhysicsBone, bone.Contents)
	}
	if !slices.Equal(m.BoneNames, []string{"root", "child"}) || !slices.Equal(m.BoneSurfaceProps, []string{"metal", "metal"}) {
		t.Errorf("bones: got names %q surface props %q", m.BoneNames, m.BoneSurfaceProps)
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

	if !slices.Equal(m.TextureNames, []string{"metal/floor", "metal/rust"}) || !slices.Equal(m.TextureDirs, []string{"models/test/"}) {
		t.Errorf("textures: got names %q dirs %q", m.TextureNames, m.TextureDirs)
	}
	if m.Name != testLongName || m.SurfaceProp != "metal" || m.KeyValues != testKeyValues {
		t.Errorf("header strings: got name %q surface prop %q key values %q", m.Name, m.SurfaceProp, m.KeyValues)
	}
	for skin, want := range []int32{0, 1} {
		if got, err := m.TextureIndexForMesh(skin, 0, 0, 0); err != nil || got != want {
			t.Errorf("skin %d texture: got %d, err %v, want %d", skin, got, err, want)
		}
	}
	if _, err := m.TextureIndexForMesh(2, 0, 0, 0); err == nil {
		t.Error("skin 2 of 2: expected an error")
	}

	if len(m.BodyParts) != 1 || len(m.BodyParts[0].Models) != 2 {
		t.Fatalf("body parts: got %+v", m.BodyParts)
	}
	if bodyPart := m.BodyParts[0]; bodyPart.Name != "body" || bodyPart.Models[0].Name != "body" || bodyPart.Models[1].Name != "blank" {
		t.Errorf("body part names: got %q with models %q, %q", bodyPart.Name, bodyPart.Models[0].Name, bodyPart.Models[1].Name)
	}
	if meshes := m.BodyParts[0].Models[0].Meshes; len(meshes) != 1 || meshes[0].Material != 0 {
		t.Errorf("model 0 meshes: got %+v", meshes)
	}
	if meshes := m.BodyParts[0].Models[1].Meshes; len(meshes) != 0 {
		t.Errorf("blank model meshes: got %+v", meshes)
	}
}

func TestReadRigging(t *testing.T) {
	m, err := ReadFromStream(bytes.NewReader(buildMdl().Bytes))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m.HitboxNames, [][]string{{"head", "head"}}) || !slices.Equal(m.HitboxSetNames, []string{"default"}) {
		t.Errorf("hitbox names: got %q in sets %q", m.HitboxNames, m.HitboxSetNames)
	}
	if len(m.Hitboxes) != 1 || len(m.Hitboxes[0]) != 2 {
		t.Fatalf("hitboxes: got %+v", m.Hitboxes)
	}
	if h := m.Hitboxes[0][0]; h.Bone != 1 || h.Group != 3 || h.BBMin != (mgl32.Vec3{-1, -2, -3}) || h.BBMax != (mgl32.Vec3{1, 2, 3}) || h.Angles != (mgl32.Vec3{0, 90, 0}) || h.CapsuleRadius != 2.5 {
		t.Errorf("hitbox 0: got %+v", h)
	}

	if len(m.Attachments) != 1 || m.AttachmentNames[0] != "muzzle" || m.Attachments[0].LocalBone != 1 || m.Attachments[0].Local.Mat3x4().Col(3) != (mgl32.Vec3{1, 2, 3}) {
		t.Errorf("attachments: got %+v named %q", m.Attachments, m.AttachmentNames)
	}

	if len(m.PoseParameters) != 1 || m.PoseParameterNames[0] != "aim_yaw" || m.PoseParameters[0].Start != -45 || m.PoseParameters[0].End != 45 {
		t.Errorf("pose parameters: got %+v named %q", m.PoseParameters, m.PoseParameterNames)
	}

	if len(m.IKChains) != 1 {
		t.Fatalf("IK chains: got %+v", m.IKChains)
	}
	if chain := m.IKChains[0]; chain.Name != "leg" || len(chain.Links) != 2 || chain.Links[1].Bone != 1 || chain.Links[1].KneeDir != (mgl32.Vec3{0, 0, 1}) {
		t.Errorf("IK chain: got %+v", chain)
	}
	if len(m.IKLocks) != 1 || m.IKLocks[0].PosWeight != 1 || m.IKLocks[0].LocalQWeight != 0.5 {
		t.Errorf("IK locks: got %+v", m.IKLocks)
	}

	if want := []IncludeModel{{Name: "models/anims.mdl"}}; !slices.Equal(m.IncludeModels, want) {
		t.Errorf("include models: got %+v, want %+v", m.IncludeModels, want)
	}

	if len(m.ProceduralBones) != 2 {
		t.Fatalf("procedural bones: got %+v", m.ProceduralBones)
	}
	quat := m.ProceduralBones[0]
	if quat.Bone != 0 || quat.Type != ProcQuatInterp || quat.QuatInterp == nil || quat.QuatInterp.Control != 1 || len(quat.QuatInterp.Triggers) != 1 ||
		quat.QuatInterp.Triggers[0].InvTolerance != 2 || quat.QuatInterp.Triggers[0].Pos != (mgl32.Vec3{1, 0, 0}) {
		t.Errorf("quat interp bone: got %+v", quat)
	}
	jiggle := m.ProceduralBones[1]
	if jiggle.Bone != 1 || jiggle.Type != ProcJiggle || jiggle.Jiggle == nil || jiggle.Jiggle.Length != 10 || jiggle.Jiggle.TipMass != 2 || jiggle.Boing == nil || jiggle.Boing.ImpactSpeed != 100 {
		t.Errorf("jiggle bone: got %+v", jiggle)
	}

	if m.Header2 == nil || !slices.Equal(m.SrcBoneTransformNames, []string{"root"}) || m.SrcBoneTransforms[0].PreTransform.Mat3x4().Col(3) != (mgl32.Vec3{0, 0, 5}) {
		t.Errorf("source bone transforms: got %+v named %q", m.SrcBoneTransforms, m.SrcBoneTransformNames)
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
		"bones past end":       func(b *testutil.Builder) { b.SetI32(160, int32(len(b.Bytes)-boneSize)) },
		"model offset wraps":   func(b *testutil.Builder) { b.SetI32(bodyPartAt+12, math.MaxInt32) },
		"length past end":      func(b *testutil.Builder) { b.SetI32(76, int32(len(b.Bytes)+1)) },
		"huge skin table":      func(b *testutil.Builder) { b.SetI32(224, math.MaxInt32) },
		"key values past end":  func(b *testutil.Builder) { b.SetI32(316, int32(len(b.Bytes))) },
		"huge hitbox count": func(b *testutil.Builder) {
			b.SetI32(int(binary.LittleEndian.Uint32(b.Bytes[176:]))+4, math.MaxInt32)
		},
		"procedural bone past end":        func(b *testutil.Builder) { b.SetI32(bonesAt+168, int32(len(b.Bytes))) },
		"source bone transforms past end": func(b *testutil.Builder) { b.SetI32(hdr2At+4, int32(len(b.Bytes))) },
		"negative IK link count": func(b *testutil.Builder) {
			b.SetI32(int(binary.LittleEndian.Uint32(b.Bytes[288:]))+8, -1)
		},
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
	animated, _ := buildAnimatedMdl()
	f.Add(animated)
	f.Add(buildFaceMdl(false))
	f.Add(buildFaceMdl(true))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := ReadFromStream(bytes.NewReader(data))
		if err != nil {
			return
		}
		_, _ = m.FlexWeights(make([]float32, len(m.FlexControllers)))
		for i, desc := range m.AnimDescs {
			for frame := range min(int(desc.NumFrames), 16) {
				_, _ = m.AnimationFrame(i, frame)
			}
		}
	})
}
