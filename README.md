# StudioModel
Golang library for loading Valve StudioModel formats (.mdl, .vtx, .vvd)

Some parts of a prop are mandatory (mdl,vvd,vtx), others are not (phy). It's up to the 
implementor to construct a StudioModel the way they want to. 

This is a collection of parsers for different formats, it has no concept of 
the filesystem structure (theoretically different StudioModel components could be located 
in different folders)



### Usage
```go

import (
	studiomodel "github.com/galaco/StudioModel"
	"github.com/galaco/StudioModel/mdl"
	"log"
)


func main() {
	// create model
	prop := studiomodel.NewStudioModel(filePath)

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