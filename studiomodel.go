package studiomodel

import (
	"github.com/galaco/studiomodel/mdl"
	"github.com/galaco/studiomodel/phy"
	"github.com/galaco/studiomodel/vtx"
	"github.com/galaco/studiomodel/vvd"
)

// type studiomodel struct {
type StudioModel struct {
	// Filename
	Filename string
	// Mdl
	Mdl *mdl.Mdl
	// Vvd
	Vvd *vvd.Vvd
	// Vtx
	Vtx *vtx.Vtx
	// Phy
	Phy *phy.Phy
}

// HasCollisionModel
func (model *StudioModel) HasCollisionModel() bool {
	return model.Phy != nil
}

// AddMdl
func (model *StudioModel) AddMdl(file *mdl.Mdl) {
	model.Mdl = file
}

// AddVvd
func (model *StudioModel) AddVvd(file *vvd.Vvd) {
	model.Vvd = file
}

// AddVtx
func (model *StudioModel) AddVtx(file *vtx.Vtx) {
	model.Vtx = file
}

// AddPhy
func (model *StudioModel) AddPhy(file *phy.Phy) {
	model.Phy = file
}

// Newstudiomodel returns a new Studiomodel
func NewStudioModel(filename string) *StudioModel {
	return &StudioModel{
		Filename: filename,
	}
}
