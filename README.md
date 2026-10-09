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

* VVD reader is stable. Fixups are exposed but not yet applied, so multi-LOD models need them resolved by the caller
* VTX reader reads all LODs. The larger strip headers used by MDL v49 are detected automatically, or can be
selected with `vtx.ReadFromStreamWithMDLVersion`
* MDL reader is usable, currently incomplete: it reads the header, bones, bone controllers, hitbox sets, animation and
sequence descriptions, textures and the body part / model / mesh tree. Names, skins, animation data and most other
sections are not read yet
* PHY reader reads every solid and its convex pieces. The key-values text that follows them is exposed as a string
but not parsed yet



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
	
	log.Println(prop)
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


