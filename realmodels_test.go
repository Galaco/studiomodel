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

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mdl") {
			return nil
		}
		name, _ := filepath.Rel(dir, path)
		t.Run(name, func(t *testing.T) {
			checkRealModel(t, strings.TrimSuffix(path, filepath.Ext(path)))
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkRealModel(t *testing.T, base string) {
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
		if err != nil || name == "" || strings.IndexFunc(name, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
			t.Errorf("mdl bone %d name: got %q, err %v", i, name, err)
		}
	}

	if data, err := readFirst(base, ".dx90.vtx", ".vtx", ".dx80.vtx", ".sw.vtx"); err == nil {
		v, err := vtx.ReadFromStream(bytes.NewReader(data))
		if err != nil {
			t.Errorf("vtx: %v", err)
		} else {
			checkVtxMatchesMdl(t, v, m)
		}
	}

	if data, err := os.ReadFile(base + ".vvd"); err == nil {
		if _, err := vvd.ReadFromStream(bytes.NewReader(data)); err != nil {
			t.Errorf("vvd: %v", err)
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
