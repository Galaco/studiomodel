[![GoDoc](https://godoc.org/github.com/Galaco/studiomodel?status.svg)](https://godoc.org/github.com/Galaco/studiomodel)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/studiomodel)](https://goreportcard.com/badge/github.com/galaco/studiomodel)
[![GolangCI](https://golangci.com/badges/github.com/galaco/studiomodel.svg)](https://golangci.com)\
[![codecov](https://codecov.io/gh/Galaco/studiomodel/branch/master/graph/badge.svg)](https://codecov.io/gh/Galaco/studiomodel)
[![CircleCI](https://circleci.com/gh/Galaco/studiomodel.svg?style=svg)](https://circleci.com/gh/Galaco/studiomodel)

# studiomodel
Golang library for loading Valve studiomodel formats (.mdl, .vtx, .vvd)

Some parts of a prop are mandatory (mdl,vvd,vtx), others are not (phy). It's up to the 
implementor to construct a studiomodel the way they want to. 

This is a collection of parsers for different formats, it has no concept of 
the filesystem structure (theoretically different StudioModel components could be located 
in different folders).

Tested against Counter Strike Source and Counter Strike Global Offensive


#### Features

* VVD reader is stable
* VTX reader is usable, only for single LOD models
* MDL reader is usable, currently incomplete (some properties not populated)
* PHY reader is usable, string data table is not supported yet



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
	prop := studiomodel.Newstudiomodel("models/error")

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


