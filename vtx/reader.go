package vtx

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/galaco/studiomodel/internal"
)

const (
	// VTXVersion is the expected file version for Source engine VTX files
	VTXVersion = 7
	// ExtendedHeaderMDLVersion is the first MDL version whose VTX strip group and strip headers carry two extra
	// topology fields (8 bytes)
	ExtendedHeaderMDLVersion = 49
	// topologyFieldsSize is the size of the topology fields added in MDL v49
	topologyFieldsSize = 8
)

// Reader
type Reader struct {
	buf []byte
	// MDLVersion is the version of the mdl this vtx belongs to. When non-zero it selects the strip group and strip
	// header layout; when zero the layout is detected from the data.
	MDLVersion int32
}

// meshRef locates the strip groups of one parsed mesh
type meshRef struct {
	mesh            *Mesh
	stripGroupStart int64
	numStripGroups  int32

	bodyPart, model, lod, index int
}

// Read parses a stream to a Vtx struct
func (reader *Reader) Read(stream io.Reader) (*Vtx, error) {
	buf, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	reader.buf = buf

	// Validate minimum file size
	headerSize := binary.Size(Header{})
	if len(reader.buf) < headerSize {
		return nil, fmt.Errorf("vtx file too small: %d bytes, expected at least %d", len(reader.buf), headerSize)
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

	// Parse everything down to mesh headers first: the strip group layout can only be detected from them
	out, meshes, err := reader.readMeshes(&header)
	if err != nil {
		return nil, err
	}

	extended := reader.MDLVersion >= ExtendedHeaderMDLVersion
	if reader.MDLVersion == 0 {
		extended = reader.detectExtendedHeaders(meshes)
	}

	for _, ref := range meshes {
		stripGroups, err := reader.readStripGroups(ref.stripGroupStart, ref.numStripGroups, extended)
		if err != nil {
			return nil, fmt.Errorf("failed to read strip groups for mesh %d, LOD %d, model %d, body part %d: %w", ref.index, ref.lod, ref.model, ref.bodyPart, err)
		}
		ref.mesh.StripGroups = stripGroups
	}

	out.MaterialReplacements, err = reader.readMaterialReplacements(&header)
	if err != nil {
		return nil, fmt.Errorf("failed to read material replacements: %w", err)
	}

	return out, nil
}

// readHeader reads studiohdr header information
func (reader *Reader) readHeader() (Header, error) {
	return internal.ReadStruct[Header](reader.buf, 0, "header")
}

// readMaterialReplacements reads each LOD's material replacements
func (reader *Reader) readMaterialReplacements(header *Header) ([][]MaterialReplacement, error) {
	if header.MaterialReplacementListOffset == 0 {
		return nil, nil
	}

	listStart := int64(header.MaterialReplacementListOffset)
	listSize := int64(binary.Size(materialReplacementListHeader{}))
	replacementSize := int64(binary.Size(materialReplacementHeader{}))
	lists, err := internal.ReadSlice[materialReplacementListHeader](reader.buf, listStart, int64(header.NumLODs), "material replacement lists")
	if err != nil {
		return nil, err
	}

	out := make([][]MaterialReplacement, len(lists))
	for lod, list := range lists {
		// Offsets are relative to the list, and names to their replacement
		replacementStart := listStart + int64(lod)*listSize + int64(list.ReplacementOffset)
		replacements, err := internal.ReadSlice[materialReplacementHeader](reader.buf, replacementStart, int64(list.NumReplacements), "material replacements")
		if err != nil {
			return nil, fmt.Errorf("LOD %d: %w", lod, err)
		}
		out[lod] = make([]MaterialReplacement, len(replacements))
		for i, replacement := range replacements {
			namePos := replacementStart + int64(i)*replacementSize + int64(replacement.NameOffset)
			name, err := internal.CString(reader.buf, namePos, 256, "material replacement name")
			if err != nil {
				return nil, fmt.Errorf("LOD %d: %w", lod, err)
			}
			out[lod][i] = MaterialReplacement{
				MaterialID: replacement.MaterialID,
				Name:       name,
			}
		}
	}

	return out, nil
}

// readMeshes parses body parts, models, LODs and meshes, and returns where each mesh's strip groups are.
// Counts of zero are valid: a blank bodygroup model has no meshes.
func (reader *Reader) readMeshes(header *Header) (*Vtx, []meshRef, error) {
	bodyPartHeaderSize := int64(binary.Size(bodyPartHeader{}))
	modelHeaderSize := int64(binary.Size(modelHeader{}))
	modelLODHeaderSize := int64(binary.Size(modelLODHeader{}))
	meshHeaderSize := int64(binary.Size(meshHeader{}))

	out := &Vtx{Header: *header}
	var meshes []meshRef

	bodyPartStart := int64(header.BodyPartOffset)
	bodyPartHeaders, err := internal.ReadSlice[bodyPartHeader](reader.buf, bodyPartStart, int64(header.NumBodyParts), "body parts")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read body parts: %w", err)
	}

	out.BodyParts = make([]BodyPart, len(bodyPartHeaders))

	// Iterate through body parts
	for i, bodyPartHeader := range bodyPartHeaders {
		modelStart := bodyPartStart + int64(i)*bodyPartHeaderSize + int64(bodyPartHeader.ModelOffset)
		modelHeaders, err := internal.ReadSlice[modelHeader](reader.buf, modelStart, int64(bodyPartHeader.NumModels), "models")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read models for body part %d: %w", i, err)
		}

		bodyPartOut := &out.BodyParts[i]
		bodyPartOut.Models = make([]Model, len(modelHeaders))

		// Iterate through models
		for j, modelHeader := range modelHeaders {
			modelLODStart := modelStart + int64(j)*modelHeaderSize + int64(modelHeader.LODOffset)
			modelLODHeaders, err := internal.ReadSlice[modelLODHeader](reader.buf, modelLODStart, int64(modelHeader.NumLODs), "model LODs")
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read LODs for model %d in body part %d: %w", j, i, err)
			}

			modelOut := &bodyPartOut.Models[j]
			modelOut.LODS = make([]ModelLOD, len(modelLODHeaders))

			// Iterate through LODs
			for k, modelLODHeader := range modelLODHeaders {
				meshStart := modelLODStart + int64(k)*modelLODHeaderSize + int64(modelLODHeader.MeshOffset)
				meshHeaders, err := internal.ReadSlice[meshHeader](reader.buf, meshStart, int64(modelLODHeader.NumMeshes), "meshes")
				if err != nil {
					return nil, nil, fmt.Errorf("failed to read meshes for LOD %d, model %d, body part %d: %w", k, j, i, err)
				}

				modelLODOut := &modelOut.LODS[k]
				modelLODOut.SwitchPoint = modelLODHeader.SwitchPoint
				modelLODOut.Meshes = make([]Mesh, len(meshHeaders))

				for l, meshHeader := range meshHeaders {
					modelLODOut.Meshes[l].Flags = meshHeader.Flags
					meshes = append(meshes, meshRef{
						mesh:            &modelLODOut.Meshes[l],
						stripGroupStart: meshStart + int64(l)*meshHeaderSize + int64(meshHeader.StripGroupHeaderOffset),
						numStripGroups:  meshHeader.NumStripGroups,
						bodyPart:        i,
						model:           j,
						lod:             k,
						index:           l,
					})
				}
			}
		}
	}

	return out, meshes, nil
}

// detectExtendedHeaders reports whether strip group and strip headers use the MDL v49 layout. The layouts only
// differ from the second strip group or strip of a mesh onwards, so the first mesh that parses under exactly one
// layout decides. When no mesh does, the layouts read identically and either is correct.
func (reader *Reader) detectExtendedHeaders(meshes []meshRef) bool {
	for _, ref := range meshes {
		_, errBase := reader.readStripGroups(ref.stripGroupStart, ref.numStripGroups, false)
		_, errExtended := reader.readStripGroups(ref.stripGroupStart, ref.numStripGroups, true)
		if (errBase == nil) != (errExtended == nil) {
			return errExtended == nil
		}
	}
	return false
}

// readStripGroups reads a mesh's strip groups with their vertices, indices and strips.
// extended selects the MDL v49 header layout.
func (reader *Reader) readStripGroups(offset int64, num int32, extended bool) ([]StripGroup, error) {
	stripGroupStride := int64(binary.Size(stripGroupHeader{}))
	stripStride := int64(binary.Size(stripHeader{}))
	if extended {
		stripGroupStride += topologyFieldsSize
		stripStride += topologyFieldsSize
	}

	stripGroupHeaders, err := internal.ReadStrided[stripGroupHeader](reader.buf, offset, int64(num), stripGroupStride, "strip groups")
	if err != nil {
		return nil, err
	}

	stripGroups := make([]StripGroup, len(stripGroupHeaders))
	for m, stripGroupHeader := range stripGroupHeaders {
		stripGroupPos := offset + int64(m)*stripGroupStride
		stripGroups[m].Flags = stripGroupHeader.Flags

		stripGroups[m].Vertexes, err = internal.ReadSlice[Vertex](reader.buf, stripGroupPos+int64(stripGroupHeader.VertOffset), int64(stripGroupHeader.NumVerts), "vertices")
		if err != nil {
			return nil, fmt.Errorf("failed to read vertices for strip group %d: %w", m, err)
		}

		stripGroups[m].Indices, err = internal.ReadSlice[uint16](reader.buf, stripGroupPos+int64(stripGroupHeader.IndexOffset), int64(stripGroupHeader.NumIndices), "indices")
		if err != nil {
			return nil, fmt.Errorf("failed to read indices for strip group %d: %w", m, err)
		}

		stripsPos := stripGroupPos + int64(stripGroupHeader.StripOffset)
		stripHeaders, err := internal.ReadStrided[stripHeader](reader.buf, stripsPos, int64(stripGroupHeader.NumStrips), stripStride, "strips")
		if err != nil {
			return nil, fmt.Errorf("failed to read strips for strip group %d: %w", m, err)
		}

		stripGroups[m].Strips = make([]Strip, len(stripHeaders))
		for s, strip := range stripHeaders {
			// Strips index into their strip group's indices and vertices
			if !withinRange(strip.IndexOffset, strip.NumIndices, stripGroupHeader.NumIndices) ||
				!withinRange(strip.VertOffset, strip.NumVerts, stripGroupHeader.NumVerts) {
				return nil, fmt.Errorf("strip %d of strip group %d lies outside the strip group's indices or vertices", s, m)
			}

			// BoneStateChangeOffset is relative to the start of this strip
			changes, err := internal.ReadSlice[BoneStateChange](reader.buf, stripsPos+int64(s)*stripStride+int64(strip.BoneStateChangeOffset), int64(strip.NumBoneStateChanges), "bone state changes")
			if err != nil {
				return nil, fmt.Errorf("failed to read bone state changes for strip %d of strip group %d: %w", s, m, err)
			}

			stripGroups[m].Strips[s] = Strip{
				NumIndices:            strip.NumIndices,
				IndexOffset:           strip.IndexOffset,
				NumVerts:              strip.NumVerts,
				VertOffset:            strip.VertOffset,
				NumBones:              strip.NumBones,
				Flags:                 strip.Flags,
				NumBoneStateChanges:   strip.NumBoneStateChanges,
				BoneStateChangeOffset: strip.BoneStateChangeOffset,
				BoneStateChanges:      changes,
			}
		}
	}

	return stripGroups, nil
}

// withinRange reports whether [offset, offset+count) lies within [0, total)
func withinRange(offset int32, count int32, total int32) bool {
	return offset >= 0 && count >= 0 && int64(offset)+int64(count) <= int64(total)
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
