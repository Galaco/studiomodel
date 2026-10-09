package studiomodel

import (
	"errors"
	"fmt"

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

// Validate checks that the vvd, vtx and phy were compiled with the mdl, by comparing their checksums
func (model *StudioModel) Validate() error {
	if model.Mdl == nil {
		return errors.New("no mdl to validate against")
	}
	want := model.Mdl.Header.Checksum
	if model.Vvd != nil && model.Vvd.Header.Checksum != want {
		return fmt.Errorf("vvd checksum %d does not match mdl checksum %d", model.Vvd.Header.Checksum, want)
	}
	if model.Vtx != nil && model.Vtx.Header.CheckSum != want {
		return fmt.Errorf("vtx checksum %d does not match mdl checksum %d", model.Vtx.Header.CheckSum, want)
	}
	if model.Phy != nil && model.Phy.Header.CheckSum != want {
		return fmt.Errorf("phy checksum %d does not match mdl checksum %d", model.Phy.Header.CheckSum, want)
	}
	return nil
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
