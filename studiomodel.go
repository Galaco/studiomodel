package studiomodel

import (
	"github.com/galaco/StudioModel/mdl"
	"github.com/galaco/StudioModel/phy"
	"github.com/galaco/StudioModel/vtx"
	"github.com/galaco/StudioModel/vvd"
)

// type StudioModel struct {
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

// NewStudioModel returns a new Studiomodel
func NewStudioModel(filename string) *StudioModel {
	return &StudioModel{
		Filename: filename,
	}
}
