[![GoDoc](https://godoc.org/github.com/Galaco/studiomodel?status.svg)](https://godoc.org/github.com/Galaco/studiomodel)
[![Go report card](https://goreportcard.com/badge/github.com/galaco/studiomodel)](https://goreportcard.com/badge/github.com/galaco/studiomodel)
[![Build Status](https://travis-ci.com/Galaco/studiomodel.svg?branch=master)](https://travis-ci.com/Galaco/studiomodel)
[![CircleCI](https://circleci.com/gh/Galaco/studiomodel/tree/master.svg?style=svg)](https://circleci.com/gh/Galaco/studiomodel/tree/master)

# studiomodel
Golang library for loading Valve studiomodel formats (.mdl, .vtx, .vvd)

Some parts of a prop are mandatory (mdl,vvd,vtx), others are not (phy). It's up to the 
implementor to construct a studiomodel the way they want to. 

This is a collection of parsers for different formats, it has no concept of 
the filesystem structure (theoretically different studiomodel components could be located 
in different folders)


##### Notice: this is very incomplete. vvd readers is stable; tested against CS:S and CS:GO. vtx reader is reliable,
only for single LOD models. mdl loader is currently imcomplete



### Usage
```go
package main

import (
	"github.com/galaco/studiomodel"
	"github.com/galaco/studiomodel/mdl"
	"log"
)


func main() {
	// create model
	prop := studiomodel.Newstudiomodel("models/error")

    // MDL
	f,err := file.Load(filePath + ".mdl") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return nil
	}
	mdlFile,err := mdl.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	prop.AddMdl(mdlFile)
	
	// VVD
	f,err = file.Load(filePath + ".vvd") // file.Load just returns (io.Reader,error)
	if err != nil {
		log.Println(err)
		return nil
	}
	vvdFile,err := vvd.ReadFromStream(f)
	if err != nil {
		log.Println(err)
		return nil
	}
	prop.AddVvd(vvdFile)
	
	log.Println(prop)
}
```


