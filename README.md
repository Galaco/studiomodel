[![GoDoc](https://godoc.org/github.com/Galaco/StudioModel?status.svg)](https://godoc.org/github.com/Galaco/StudioModel)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/studiomodel)](https://goreportcard.com/badge/github.com/galaco/studiomodel)
[![GolangCI](https://golangci.com/badges/github.com/galaco/StudioModel.svg)](https://golangci.com)
[![Build Status](https://travis-ci.com/Galaco/StudioModel.svg?branch=master)](https://travis-ci.com/Galaco/StudioModel)
[![codecov](https://codecov.io/gh/Galaco/StudioModel/branch/master/graph/badge.svg)](https://codecov.io/gh/Galaco/StudioModel)
[![CircleCI](https://circleci.com/gh/Galaco/StudioModel.svg?style=svg)](https://circleci.com/gh/Galaco/StudioModel)

# StudioModel
Golang library for loading Valve StudioModel formats (.mdl, .vtx, .vvd)

Some parts of a prop are mandatory (mdl,vvd,vtx), others are not (phy). It's up to the 
implementor to construct a StudioModel the way they want to. 

This is a collection of parsers for different formats, it has no concept of 
the filesystem structure (theoretically different StudioModel components could be located 
in different folders)


##### Notice: this is very incomplete. vvd readers is stable; tested against CS:S and CS:GO. vtx reader is reliable,
only for single LOD models. mdl loader is currently incomplete.



### Usage
```go
package main

import (
	studiomodel "github.com/galaco/StudioModel"
	"github.com/galaco/StudioModel/mdl"
	"github.com/galaco/StudioModel/vvd"
	"log"
	"os"
)


func main() {
	filePath := "foo/prop" //
	
	// create model
	prop := studiomodel.NewStudioModel(filePath)

    // MDL
	f,err := file.Load(filePath + ".mdl") // file.Load just returns (io.Reader,error)
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
	f,err = file.Load(filePath + ".vvd") // file.Load just returns (io.Reader,error)
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
	
	log.Println(prop)
}
```


