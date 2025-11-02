package vtx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/galaco/studiomodel/internal"
	"io"
	"unsafe"
)

const (
	// VTXVersion is the expected file version for Source engine VTX files
	VTXVersion = 7
)

// Reader
type Reader struct {
	buf []byte
}

// Read parses a stream to a Vtx struct
func (reader *Reader) Read(stream io.Reader) (*Vtx, error) {
	byteBuf := bytes.Buffer{}
	_, err := byteBuf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	reader.buf = byteBuf.Bytes()

	// Validate minimum file size
	if len(reader.buf) < int(unsafe.Sizeof(header{})) {
		return nil, fmt.Errorf("vtx file too small: %d bytes, expected at least %d", len(reader.buf), unsafe.Sizeof(header{}))
	}

	// Read header
	header, err := reader.readHeader()
	if err != nil {
		return nil, fmt.Errorf("failed to read VTX header: %w", err)
	}

	// Validate version
	if header.Version != VTXVersion {
		return nil, fmt.Errorf("unsupported VTX version: got %d, expected %d", header.Version, VTXVersion)
	}

	// Validate counts
	if header.NumLODs < 0 || header.NumBodyParts < 0 {
		return nil, fmt.Errorf("VTX header contains negative counts: NumLODs=%d NumBodyParts=%d", header.NumLODs, header.NumBodyParts)
	}

	// Validate offsets
	if header.BodyPartOffset < 0 || int(header.BodyPartOffset) > len(reader.buf) {
		return nil, fmt.Errorf("VTX body part offset %d out of bounds (file size %d)", header.BodyPartOffset, len(reader.buf))
	}

	out := Vtx{}

	// Parse body parts
	bodyPartHeaderSize := internal.SizeOf(&bodyPartHeader{})
	bodyPartStart := header.BodyPartOffset

	bodyPartHeaders, err := reader.readBodyParts(bodyPartStart, header.NumBodyParts)
	if err != nil {
		return nil, fmt.Errorf("failed to read body parts: %w", err)
	}

	out.BodyParts = make([]BodyPart, len(bodyPartHeaders))

	// Iterate through body parts
	for i, bodyPartHeader := range bodyPartHeaders {
		bodyPartOut := BodyPart{}
		bodyPartPos := bodyPartStart + (int32(i) * bodyPartHeaderSize)
		modelStart := bodyPartPos + bodyPartHeader.ModelOffset

		// Parse models
		modelHeaderSize := internal.SizeOf(&modelHeader{})
		modelHeaders, err := reader.readModels(modelStart, bodyPartHeader.NumModels)
		if err != nil {
			return nil, fmt.Errorf("failed to read models for body part %d: %w", i, err)
		}

		bodyPartOut.Models = make([]Model, len(modelHeaders))

		// Iterate through models
		for j, modelHeader := range modelHeaders {
			modelOut := Model{}
			modelPos := modelStart + (int32(j) * modelHeaderSize)
			modelLODStart := modelPos + modelHeader.LODOffset

			// Parse model LODs
			modelLODHeaderSize := internal.SizeOf(&modelLODHeader{})
			modelLODHeaders, err := reader.readModelLODs(modelLODStart, modelHeader.NumLODs)
			if err != nil {
				return nil, fmt.Errorf("failed to read LODs for model %d in body part %d: %w", j, i, err)
			}

			modelOut.LODS = make([]ModelLOD, len(modelLODHeaders))

			// Iterate through LODs
			for k, modelLODHeader := range modelLODHeaders {
				modelLODOut := ModelLOD{}
				modelLODPos := modelLODStart + (int32(k) * modelLODHeaderSize)
				meshStart := modelLODPos + modelLODHeader.MeshOffset

				// Parse meshes
				meshHeaderSize := int32(9) // VTX ignores trailing byte 4-byte alignment
				meshHeaders, err := reader.readMeshes(meshStart, modelLODHeader.NumMeshes)
				if err != nil {
					return nil, fmt.Errorf("failed to read meshes for LOD %d, model %d, body part %d: %w", k, j, i, err)
				}

				modelLODOut.Meshes = make([]Mesh, len(meshHeaders))

				// Iterate through meshes
				for l, meshHeader := range meshHeaders {
					meshOut := Mesh{}
					meshPos := meshStart + (int32(l) * meshHeaderSize)
					stripGroupStart := meshPos + meshHeader.StripGroupHeaderOffset

					// Parse strip groups
					stripGroupHeaderSize := int32(25) // VTX ignores trailing byte 4-byte alignment
					stripGroupHeaders, err := reader.readStripGroups(stripGroupStart, meshHeader.NumStripGroups)
					if err != nil {
						return nil, fmt.Errorf("failed to read strip groups for mesh %d, LOD %d, model %d, body part %d: %w", l, k, j, i, err)
					}

					meshOut.StripGroups = make([]StripGroup, len(stripGroupHeaders))

					// Iterate through strip groups
					for m, stripGroupHeader := range stripGroupHeaders {
						stripGroupOut := StripGroup{}
						stripGroupPos := stripGroupStart + (int32(m) * stripGroupHeaderSize)

						// Read vertices, indices, and strips for this strip group
						stripGroupOut.Vertexes, err = reader.readVertices(stripGroupPos+stripGroupHeader.VertOffset, stripGroupHeader.NumVerts)
						if err != nil {
							return nil, fmt.Errorf("failed to read vertices for strip group %d: %w", m, err)
						}

						stripGroupOut.Indices, err = reader.readIndices(stripGroupPos+stripGroupHeader.IndexOffset, stripGroupHeader.NumIndices)
						if err != nil {
							return nil, fmt.Errorf("failed to read indices for strip group %d: %w", m, err)
						}

						stripGroupOut.Strips, err = reader.readStrips(stripGroupPos+stripGroupHeader.StripOffset, stripGroupHeader.NumStrips)
						if err != nil {
							return nil, fmt.Errorf("failed to read strips for strip group %d: %w", m, err)
						}

						meshOut.StripGroups[m] = stripGroupOut
					}

					modelLODOut.Meshes[l] = meshOut
				}

				modelOut.LODS[k] = modelLODOut
			}

			bodyPartOut.Models[j] = modelOut
		}

		out.BodyParts[i] = bodyPartOut
	}

	return &out, nil
}

// readHeader reads studiohdr header information
func (reader *Reader) readHeader() (header, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(reader.buf[:headerSize]), binary.LittleEndian, &header)

	return header, err
}

// readBodyParts
func (reader *Reader) readBodyParts(offset int32, num int32) ([]bodyPartHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]bodyPartHeader, 0), errors.New("body part data out of bounds")
	}
	ret := make([]bodyPartHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readModels
func (reader *Reader) readModels(offset int32, num int32) ([]modelHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelHeader, 0), errors.New("model data out of bounds")
	}
	ret := make([]modelHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readModelLODs
func (reader *Reader) readModelLODs(offset int32, num int32) ([]modelLODHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelLODHeader, 0), errors.New("model lod data out of bounds")
	}
	ret := make([]modelLODHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readMeshes
func (reader *Reader) readMeshes(offset int32, num int32) ([]meshHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]meshHeader, 0), errors.New("mesh data out of bounds")
	}
	ret := make([]meshHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readStripGroups
func (reader *Reader) readStripGroups(offset int32, num int32) ([]stripGroupHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]stripGroupHeader, 0), errors.New("strip group data out of bounds")
	}
	ret := make([]stripGroupHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readIndices
func (reader *Reader) readIndices(offset int32, num int32) ([]uint16, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]uint16, 0), errors.New("indices data out of bounds")
	}
	ret := make([]uint16, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readVertices
func (reader *Reader) readVertices(offset int32, num int32) ([]Vertex, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Vertex, 0), errors.New("vertex data out of bounds")
	}
	ret := make([]Vertex, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readStrips
func (reader *Reader) readStrips(offset int32, num int32) ([]Strip, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Strip, 0), errors.New("strip data out of bounds")
	}
	ret := make([]Strip, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func isPropertyValid(offset int32, num int32, bufferSize int) bool {
	if num < 1 || offset < 1 {
		return false
	}
	// Actually check if the data fits within the buffer
	// Note: This is a conservative check that doesn't know struct size,
	// but at least validates the offset is reasonable
	if int(offset) >= bufferSize {
		return false
	}
	return true
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
