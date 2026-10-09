package studiomodel_test

import (
	"bytes"
	"encoding/binary"
	"io/fs"
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

	var windings windingStats
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mdl") {
			return nil
		}
		name, _ := filepath.Rel(dir, path)
		t.Run(name, func(t *testing.T) {
			checkRealModel(t, strings.TrimSuffix(path, filepath.Ext(path)), &windings)
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Some assets ship normals that disagree with their faces, but if many LODs do, vertices are being mismatched
	if windings.lods > 0 && float64(windings.mismatched) > 0.01*float64(windings.lods) {
		t.Errorf("%d of %d LODs have triangles that disagree with their vertex normals", windings.mismatched, windings.lods)
	}
}

// windingStats counts LODs whose triangles disagree with their vertex normals
type windingStats struct {
	lods, mismatched int
}

func checkRealModel(t *testing.T, base string, windings *windingStats) {
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

	model := studiomodel.NewStudioModel(base)
	model.AddMdl(m)

	if data, err := readFirst(base, ".dx90.vtx", ".vtx", ".dx80.vtx", ".sw.vtx"); err == nil {
		v, err := vtx.ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Errorf("vtx: %v", err)
		} else {
			checkVtxMatchesMdl(t, v, m)
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
		checkLODs(t, model, windings)
	}
}

// checkLODs builds every LOD with every skin. Triangles that use the wrong vertices face random ways, whereas Source
// winds triangles clockwise against their vertex normals.
func checkLODs(t *testing.T, model *studiomodel.StudioModel, windings *windingStats) {
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
			windings.lods++
			if float64(against) < 0.8*float64(total) {
				windings.mismatched++
				// Fixups are where vertex order most easily goes wrong, so models with them must not mismatch
				if len(model.Vvd.Fixups) > 0 {
					t.Errorf("LOD %d: only %d of %d triangles wind against their vertex normals", lod, against, total)
				} else {
					t.Logf("LOD %d: only %d of %d triangles wind against their vertex normals", lod, against, total)
				}
			}
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

// checkPhy checks that each ledge's own size field agrees with what the reader found: the ledge header, its
// triangles, and the distinct points they use, 16 bytes each
func checkPhy(t *testing.T, p *phy.Phy) {
	if len(p.Solids) != int(p.Header.SolidCount) {
		t.Errorf("phy has %d solids, header says %d", len(p.Solids), p.Header.SolidCount)
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
