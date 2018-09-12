package studiomodel

import (
	"github.com/galaco/StudioModel/mdl"
	"github.com/galaco/StudioModel/phy"
	"github.com/galaco/StudioModel/vtx"
	"github.com/galaco/StudioModel/vvd"
)

type StudioModel struct {
	Filename string
	Mdl      *mdl.Mdl
	Vvd      *vvd.Vvd
	Vtx      *vtx.Vtx
	Phy      *phy.Phy
}

func (model *StudioModel) HasCollisionModel() bool {
	return model.Phy != nil
}

func (model *StudioModel) AddMdl(file *mdl.Mdl) {
	model.Mdl = file
}

func (model *StudioModel) AddVvd(file *vvd.Vvd) {
	model.Vvd = file
}

func (model *StudioModel) AddVtx(file *vtx.Vtx) {
	model.Vtx = file
}

func (model *StudioModel) AddPhy(file *phy.Phy) {
	model.Phy = file
}

func NewStudioModel(filename string) *StudioModel {
	return &StudioModel{
		Filename: filename,
	}
}
