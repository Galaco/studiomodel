package studiomodel_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/galaco/studiomodel"
	"github.com/galaco/studiomodel/internal"
	"github.com/galaco/studiomodel/mdl"
	"github.com/galaco/studiomodel/phy"
	"github.com/galaco/studiomodel/vtx"
	"github.com/galaco/studiomodel/vvd"
)

// TestRealModels parses every .mdl under $STUDIOMODEL_TESTDATA, along with the .vvd, .vtx and .phy beside it.
// Valve and third-party models can't be committed, so this only runs when pointed at a local folder of them.
func TestRealModels(t *testing.T) {
	dir := os.Getenv("STUDIOMODEL_TESTDATA")
	if dir == "" {
		t.Skip("set STUDIOMODEL_TESTDATA to a folder of models to run")
	}

	r := rates{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mdl") {
			return nil
		}
		name, _ := filepath.Rel(dir, path)
		t.Run(name, func(t *testing.T) {
			checkRealModel(t, dir, strings.TrimSuffix(path, filepath.Ext(path)), r)
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for name, c := range r {
		if c.failed > 1 && c.failed*50 > c.checked {
			t.Errorf("%s: %d of %d failed", name, c.failed, c.checked)
		}
	}
}

// rates counts how often each check that real assets occasionally fail does fail. Valve's own models have a few
// mirrored, double-sided or dangling parts; a parser bug fails such a check far more often than the 2% allowed.
type rates map[string]*rate

type rate struct {
	checked, failed int
}

func (r rates) record(t *testing.T, check string, ok bool, format string, args ...any) {
	c := r[check]
	if c == nil {
		c = &rate{}
		r[check] = c
	}
	c.checked++
	if !ok {
		c.failed++
		t.Logf(check+": "+format, args...)
	}
}

func checkRealModel(t *testing.T, root string, base string, r rates) {
	buf, err := os.ReadFile(base + ".mdl")
	if err != nil {
		t.Fatal(err)
	}
	m, err := mdl.ReadFromStream(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("mdl: %v", err)
	}

	// Every bone's name resolves only if bones are read at the right size
	boneSize := int64(binary.Size(mdl.Bone{}))
	for i, bone := range m.Bones {
		offset := int64(m.Header.BoneOffset) + int64(i)*boneSize + int64(bone.NameIndex)
		name, err := internal.CString(buf, offset, 256, "bone name")
		if err != nil || name == "" || strings.IndexFunc(name, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 || name != m.BoneNames[i] {
			t.Errorf("mdl bone %d name: got %q (%q), err %v", i, name, m.BoneNames[i], err)
		}
	}
	for skin, family := range m.SkinFamilies {
		for slot, texture := range family {
			if texture < 0 || int(texture) >= len(m.TextureNames) {
				t.Errorf("mdl skin %d slot %d: texture %d of %d", skin, slot, texture, len(m.TextureNames))
			}
		}
	}

	checkRigging(t, m)
	loadAnimBlocks(t, root, base, m)
	checkAnimations(t, m, r)
	checkFaces(t, m)

	model := studiomodel.NewStudioModel(base)
	model.AddMdl(m)

	if data, err := readFirst(base, ".dx90.vtx", ".vtx", ".dx80.vtx", ".sw.vtx"); err == nil {
		v, err := vtx.ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Errorf("vtx: %v", err)
		} else {
			checkVtxMatchesMdl(t, v, m)
			checkBoneStateChanges(t, v, m)
			model.AddVtx(v)
		}
	}

	if data, err := os.ReadFile(base + ".vvd"); err == nil {
		v, err := vvd.ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Errorf("vvd: %v", err)
		} else {
			model.AddVvd(v)
		}
	}

	if data, err := os.ReadFile(base + ".phy"); err == nil {
		p, err := phy.ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Errorf("phy: %v", err)
		} else {
			checkPhy(t, p)
		}
	}

	// The engine rejects a vvd or vtx from another compile. Some shipped models do have a mismatched phy, so it is
	// not checked.
	if err := model.Validate(); err != nil {
		t.Error(err)
	}

	if model.Vvd != nil && model.Vtx != nil {
		checkLODs(t, model, r)
	}
}

// loadAnimBlocks loads the .ani file a model keeps animations in, from the game folder the model is in or beside it
func loadAnimBlocks(t *testing.T, root string, base string, m *mdl.Mdl) {
	if m.AnimBlockName == "" {
		return
	}
	name := filepath.FromSlash(strings.ReplaceAll(m.AnimBlockName, "\\", "/"))
	data, err := readFirst("", filepath.Join(root, name), filepath.Join(filepath.Dir(base), filepath.Base(name)))
	if err != nil {
		return
	}
	if err := m.LoadAnimBlocks(bytes.NewReader(data)); err != nil {
		t.Errorf("%s: %v", m.AnimBlockName, err)
	}
}

// checkLODs builds every LOD with every skin. Triangles that use the wrong vertices face random ways, whereas Source
// winds triangles consistently: clockwise against their vertex normals, or the other way round for mirrored models.
func checkLODs(t *testing.T, model *studiomodel.StudioModel, r rates) {
	for lod := range int(model.Vtx.Header.NumLODs) {
		for skin := range len(model.Mdl.SkinFamilies) {
			geometry, err := model.BuildLOD(lod, skin)
			if err != nil {
				t.Errorf("LOD %d skin %d: %v", lod, skin, err)
				continue
			}
			if skin != 0 {
				continue
			}

			against, total := 0, 0
			for _, mesh := range geometry.Meshes {
				for i := 0; i+2 < len(mesh.Indices); i += 3 {
					a, b, c := geometry.Vertices[mesh.Indices[i]], geometry.Vertices[mesh.Indices[i+1]], geometry.Vertices[mesh.Indices[i+2]]
					face := b.Position.Sub(a.Position).Cross(c.Position.Sub(a.Position))
					if face.Len() < 1e-6 {
						continue
					}
					total++
					if face.Dot(a.Normal.Add(b.Normal).Add(c.Normal)) < 0 {
						against++
					}
				}
			}
			if total == 0 {
				continue
			}
			// Fixups are where vertex order most easily goes wrong, so models with them are counted apart
			check := "triangle winding"
			if len(model.Vvd.Fixups) > 0 {
				check = "triangle winding with fixups"
			}
			consistent := against*5 >= total*4 || against*5 <= total
			r.record(t, check, consistent, "LOD %d: %d of %d triangles wind against their vertex normals", lod, against, total)
		}
	}
}

// checkVtxMatchesMdl checks that every LOD of every vtx model has the same meshes as the mdl model
func checkVtxMatchesMdl(t *testing.T, v *vtx.Vtx, m *mdl.Mdl) {
	if len(v.BodyParts) != len(m.BodyParts) {
		t.Errorf("vtx has %d body parts, mdl has %d", len(v.BodyParts), len(m.BodyParts))
		return
	}
	for i, bodyPart := range v.BodyParts {
		if len(bodyPart.Models) != len(m.BodyParts[i].Models) {
			t.Errorf("body part %d: vtx has %d models, mdl has %d", i, len(bodyPart.Models), len(m.BodyParts[i].Models))
			continue
		}
		for j, model := range bodyPart.Models {
			for k, lod := range model.LODS {
				if want := len(m.BodyParts[i].Models[j].Meshes); len(lod.Meshes) != want {
					t.Errorf("body part %d model %d LOD %d: vtx has %d meshes, mdl has %d", i, j, k, len(lod.Meshes), want)
				}
			}
		}
	}
}

// checkRigging checks that hitboxes, attachments, IK links and procedural bones refer to bones that exist
func checkRigging(t *testing.T, m *mdl.Mdl) {
	bone := func(what string, i int32) {
		if i < 0 || int(i) >= len(m.Bones) {
			t.Errorf("%s: bone %d of %d", what, i, len(m.Bones))
		}
	}
	for i, set := range m.Hitboxes {
		for j, hitbox := range set {
			bone(fmt.Sprintf("hitbox set %d hitbox %d", i, j), hitbox.Bone)
		}
	}
	for i, attachment := range m.Attachments {
		bone(fmt.Sprintf("attachment %d (%s)", i, m.AttachmentNames[i]), attachment.LocalBone)
	}
	for i, chain := range m.IKChains {
		for j, link := range chain.Links {
			bone(fmt.Sprintf("IK chain %d link %d", i, j), link.Bone)
		}
	}
	for _, proc := range m.ProceduralBones {
		if proc.Jiggle != nil && (proc.Jiggle.Length <= 0 || math.IsInf(float64(proc.Jiggle.Length), 0)) {
			t.Errorf("jiggle bone %s: length %v", m.BoneNames[proc.Bone], proc.Jiggle.Length)
		}
	}
}

// checkAnimations decodes every frame of every animation, unless it is in an .ani file that isn't there, and checks
// that sequences refer to animations, sequences and bones that exist
func checkAnimations(t *testing.T, m *mdl.Mdl, r rates) {
	for i, desc := range m.AnimDescs {
		if _, err := m.AnimationFrame(i, 0); err != nil && strings.Contains(err.Error(), "not loaded") {
			continue
		}
		for frame := range int(desc.NumFrames) {
			if _, err := m.AnimationFrame(i, frame); err != nil {
				t.Errorf("%s: %v", m.AnimDescNames[i], err)
				break
			}
		}
		for _, movement := range m.AnimMovements[i] {
			if movement.EndFrame < 0 || movement.EndFrame > desc.NumFrames {
				t.Errorf("%s: movement ends at frame %d of %d", m.AnimDescNames[i], movement.EndFrame, desc.NumFrames)
			}
		}
	}

	// Included models add their animations and sequences after this model's, so only models without them are checked
	local := len(m.IncludeModels) == 0
	for i, seq := range m.Sequences {
		name := m.SequenceLabels[i]
		for _, anim := range seq.Animations {
			r.record(t, "sequence animations", anim >= 0 && (!local || int(anim) < len(m.AnimDescs)), "sequence %s: animation %d of %d", name, anim, len(m.AnimDescs))
		}
		for _, layer := range seq.AutoLayers {
			r.record(t, "sequence layers", layer.Sequence >= 0 && (!local || int(layer.Sequence) < len(m.SequenceDescs)), "sequence %s: layer of sequence %d of %d", name, layer.Sequence, len(m.SequenceDescs))
		}
		for _, event := range seq.Events {
			if event.Cycle < 0 || event.Cycle > 1 {
				t.Errorf("sequence %s: event at cycle %v", name, event.Cycle)
			}
		}
		if seq.BoneWeights != nil && len(seq.BoneWeights) != len(m.Bones) {
			t.Errorf("sequence %s: %d bone weights for %d bones", name, len(seq.BoneWeights), len(m.Bones))
		}
	}
}

// checkFaces runs the flex rules with every controller at its minimum and maximum, and checks that flexes, eyeballs
// and mouths refer to flex descriptions, vertices and bones that exist
func checkFaces(t *testing.T, m *mdl.Mdl) {
	for _, extreme := range []string{"minimum", "maximum"} {
		values := make([]float32, len(m.FlexControllers))
		for i, c := range m.FlexControllers {
			values[i] = c.Min
			if extreme == "maximum" {
				values[i] = c.Max
			}
		}
		weights, err := m.FlexWeights(values)
		if err != nil {
			if !strings.Contains(err.Error(), "not supported") {
				t.Errorf("flex weights at controller %s: %v", extreme, err)
			}
			break
		}
		for i, w := range weights {
			if math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) {
				t.Errorf("flex weights at controller %s: %s is %v", extreme, m.FlexDescs[i], w)
			}
		}
	}

	flexDesc := func(what string, i int32) {
		if i < -1 || int(i) >= len(m.FlexDescs) {
			t.Errorf("%s: flex description %d of %d", what, i, len(m.FlexDescs))
		}
	}
	for _, bodyPart := range m.BodyParts {
		for _, model := range bodyPart.Models {
			for k, flexes := range model.Flexes {
				for _, flex := range flexes {
					flexDesc("mesh flex", flex.Header.FlexDesc)
					for _, v := range flex.VertAnims {
						if int32(v.Index) >= model.Meshes[k].NumVertices || math.IsNaN(float64(v.Delta.Len())) || math.IsInf(float64(v.Delta.Len()), 0) {
							t.Errorf("mesh flex: vertex %d of %d moves by %v", v.Index, model.Meshes[k].NumVertices, v.Delta)
						}
					}
				}
			}
			for i, eyeball := range model.Eyeballs {
				if eyeball.Bone < 0 || int(eyeball.Bone) >= len(m.Bones) || eyeball.Radius <= 0 {
					t.Errorf("eyeball %d: bone %d of %d, radius %v", i, eyeball.Bone, len(m.Bones), eyeball.Radius)
				}
				flexDesc("eyeball upper lid", eyeball.UpperLidFlexDesc)
				flexDesc("eyeball lower lid", eyeball.LowerLidFlexDesc)
			}
		}
	}
	for i, mouth := range m.Mouths {
		if mouth.Bone < 0 || int(mouth.Bone) >= len(m.Bones) {
			t.Errorf("mouth %d: bone %d of %d", i, mouth.Bone, len(m.Bones))
		}
		flexDesc("mouth", mouth.FlexDesc)
	}
}

// checkBoneStateChanges checks that strips only put bones that exist into hardware slots
func checkBoneStateChanges(t *testing.T, v *vtx.Vtx, m *mdl.Mdl) {
	for _, bodyPart := range v.BodyParts {
		for _, model := range bodyPart.Models {
			for _, lod := range model.LODS {
				for _, mesh := range lod.Meshes {
					for _, group := range mesh.StripGroups {
						for _, strip := range group.Strips {
							for _, change := range strip.BoneStateChanges {
								if change.NewBoneID < 0 || int(change.NewBoneID) >= len(m.Bones) {
									t.Errorf("vtx bone state change: bone %d of %d", change.NewBoneID, len(m.Bones))
									return
								}
							}
						}
					}
				}
			}
		}
	}
}

// checkPhy checks that each ledge's own size field agrees with what the reader found: the ledge header, its
// triangles, and the distinct points they use, 16 bytes each
func checkPhy(t *testing.T, p *phy.Phy) {
	if len(p.Solids) != int(p.Header.SolidCount) {
		t.Errorf("phy has %d solids, header says %d", len(p.Solids), p.Header.SolidCount)
	}

	// The key-values text describes each solid once, and constrains solids that exist
	solids, err := p.SolidInfos()
	if err != nil {
		t.Errorf("phy solids: %v", err)
	}
	indices := map[int]bool{}
	for _, solid := range solids {
		if solid.Index < 0 || solid.Index >= len(p.Solids) || indices[solid.Index] {
			t.Errorf("phy solid %q: index %d of %d, or repeated", solid.Name, solid.Index, len(p.Solids))
		}
		indices[solid.Index] = true
	}
	if len(solids) != len(p.Solids) {
		t.Errorf("phy text describes %d solids, file has %d", len(solids), len(p.Solids))
	}
	constraints, err := p.RagdollConstraints()
	if err != nil {
		t.Errorf("phy ragdoll constraints: %v", err)
	}
	for _, c := range constraints {
		if c.Parent < 0 || c.Parent >= len(p.Solids) || c.Child < 0 || c.Child >= len(p.Solids) {
			t.Errorf("phy ragdoll constraint %d -> %d: of %d solids", c.Parent, c.Child, len(p.Solids))
		}
	}
	if _, err := p.CollisionPairs(); err != nil {
		t.Errorf("phy collision pairs: %v", err)
	}

	for i, solid := range p.Solids {
		for j, ledge := range solid.Ledges {
			points := map[uint16]bool{}
			for _, tri := range ledge.Triangles {
				points[tri.V1], points[tri.V2], points[tri.V3] = true, true, true
			}
			size := int(uint32(ledge.Header.Flags)>>8) * 16
			if want := 16 * (1 + len(ledge.Triangles) + len(points)); size != want {
				t.Errorf("phy solid %d ledge %d: size field says %d bytes, found %d", i, j, size, want)
			}
		}
	}
}

// readFirst reads the first of base+suffix that exists
func readFirst(base string, suffixes ...string) ([]byte, error) {
	var err error
	for _, suffix := range suffixes {
		var data []byte
		if data, err = os.ReadFile(base + suffix); err == nil {
			return data, nil
		}
	}
	return nil, err
}
