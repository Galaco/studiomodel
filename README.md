[![GoDoc](https://godoc.org/github.com/Galaco/studiomodel?status.svg)](https://godoc.org/github.com/Galaco/studiomodel)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/studiomodel)](https://goreportcard.com/badge/github.com/galaco/studiomodel)
[![Tests](https://github.com/Galaco/studiomodel/actions/workflows/test.yml/badge.svg)](https://github.com/Galaco/studiomodel/actions/workflows/test.yml)

# studiomodel
Golang library for loading Valve studiomodel formats (.mdl, .vtx, .vvd)

Some parts of a prop are mandatory (mdl,vvd,vtx), others are not (phy). It's up to the 
implementor to construct a studiomodel the way they want to. 

This is a collection of parsers for different formats, it has no concept of 
the filesystem structure (theoretically different StudioModel components could be located 
in different folders).

Tested against Counter Strike Source and Counter Strike Global Offensive


#### Features

* VVD reader is stable. `VerticesForRootLOD` applies fixups, giving vertices in the order mdl and vtx indices expect
* VTX reader reads all LODs, with their switch points, mesh and strip group flags, bone state changes and material
replacements. The larger strip headers used by MDL v49 are detected automatically, or can be selected with
`vtx.ReadFromStreamWithMDLVersion`
* MDL reader reads the header and secondary header, bones and their procedural
rules (axis and quaternion interpolation, aim-at, jiggle), bone controllers, hitboxes, attachments, pose parameters,
IK chains and locks, included models, textures, skins and the body part / model / mesh tree, with their names. It
decodes animation frames into bone poses with `AnimationFrame` (load an .ani file first with `LoadAnimBlocks` for
animations stored there), and reads root motion, IK rules, sequence details (events, blend grids, layers, bone
weights) and the transition graph. For faces it reads flex descriptions, controllers and their UI, flex rules (run
them with `FlexWeights`), mesh flexes with their vertex deltas, eyeballs, mouths and bone flex drivers
* PHY reader reads every solid and its convex pieces, including the older layout without a "VPHY" header, and
parses the key-values text that follows them into solid descriptions, ragdoll constraints and collision pairs.
`phy.ToSource` converts vertices from IVP to Source space
* `StudioModel.BuildLOD` combines the mdl, vvd and vtx into triangle lists per mesh for a LOD and skin, and
`StudioModel.Validate` checks the files' checksums match

### Usage
```go
package main

import (
	"github.com/galaco/studiomodel"
	"github.com/galaco/studiomodel/mdl"
	"github.com/galaco/studiomodel/phy"
	"github.com/galaco/studiomodel/vtx"
	"github.com/galaco/studiomodel/vvd"
	"log"
	"os"
)


func main() {
	filePath := "foo/prop" //
	
	// create model
	prop := studiomodel.NewStudioModel("models/error")

    // MDL
	f,err := os.Open(filePath + ".mdl") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return
	}
	mdlFile,err := mdl.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return
	}
	prop.AddMdl(mdlFile)

	// VVD
	f,err = os.Open(filePath + ".vvd") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return
	}
	vvdFile,err := vvd.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return
	}
	prop.AddVvd(vvdFile)

	// VTX
	f,err = os.Open(filePath + ".vtx") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return
	}
	vtxFile,err := vtx.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return
	}
	prop.AddVtx(vtxFile)

	// PHY
	f,err = os.Open(filePath + ".phy") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return
	}
	phyFile,err := phy.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return
	}
	prop.AddPhy(phyFile)

	if err := prop.Validate(); err != nil {
		log.Println(err)
		return
	}

	// Triangles for every mesh of the most detailed LOD, with skin 0
	lod, err := prop.BuildLOD(0, 0)
	if err != nil {
		log.Println(err)
		return
	}
	for _, mesh := range lod.Meshes {
		log.Println(mesh.Material, len(mesh.Indices)/3, "triangles")
	}
}
```

### Testing

```bash
go test ./...
```

Real game models can't be committed, so to also parse a local folder of them (each `.mdl` alongside its `.vvd`,
`.vtx` and `.phy`):

```bash
STUDIOMODEL_TESTDATA=/path/to/models go test -run TestRealModels .
```


