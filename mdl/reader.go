package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/galaco/studiomodel/internal"
)

const (
	// MDLMagicNumber is the expected file ID ("IDST" in little-endian)
	MDLMagicNumber = 0x54534449
	// MDLMinVersion is the minimum supported MDL version
	MDLMinVersion = 44
	// MDLMaxVersion is the maximum supported MDL version
	MDLMaxVersion = 49
)

// Reader is a parser for mdl files
type Reader struct {
}

// Read parses the passed stream and returns an Mdl
func (reader *Reader) Read(stream io.Reader) (*Mdl, error) {
	buf, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	// Validate minimum file size
	headerSize := binary.Size(Studiohdr{})
	if len(buf) < headerSize {
		return nil, fmt.Errorf("mdl file too small: %d bytes, expected at least %d", len(buf), headerSize)
	}

	header, err := reader.readHeader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read MDL header: %w", err)
	}

	// Validate magic number
	if header.Id != MDLMagicNumber {
		return nil, fmt.Errorf("invalid MDL magic number: got 0x%08X, expected 0x%08X", header.Id, MDLMagicNumber)
	}

	// Validate version
	if header.Version < MDLMinVersion || header.Version > MDLMaxVersion {
		return nil, fmt.Errorf("unsupported MDL version: got %d, expected between %d and %d", header.Version, MDLMinVersion, MDLMaxVersion)
	}

	// A file shorter than the header claims has been truncated. Trailing bytes are tolerated.
	if int64(header.DataLength) > int64(len(buf)) {
		return nil, fmt.Errorf("MDL data length mismatch: header says %d, file is %d bytes", header.DataLength, len(buf))
	}

	//if header.StudioHDR2Index > 0 {
	//	// TODO: reader header2
	//}

	// Read all properties. Counts and offsets are bounds checked before anything is allocated.
	bones, err := internal.ReadSlice[Bone](buf, int64(header.BoneOffset), int64(header.BoneCount), "bones")
	if err != nil {
		return nil, err
	}
	boneNames, err := readNames(buf, header.BoneOffset, bones, func(b Bone) int32 { return b.NameIndex }, "bone")
	if err != nil {
		return nil, err
	}
	boneSurfaceProps, err := readNames(buf, header.BoneOffset, bones, func(b Bone) int32 { return b.SurfacePropIndex }, "bone surface prop")
	if err != nil {
		return nil, err
	}

	boneControllers, err := internal.ReadSlice[BoneController](buf, int64(header.BoneControllerOffset), int64(header.BoneControllerCount), "bone controllers")
	if err != nil {
		return nil, err
	}

	hitboxSets, err := internal.ReadSlice[HitboxSet](buf, int64(header.HitboxOffset), int64(header.HitboxCount), "hitbox sets")
	if err != nil {
		return nil, err
	}
	hitboxSetNames, err := readNames(buf, header.HitboxOffset, hitboxSets, func(h HitboxSet) int32 { return h.NameIndex }, "hitbox set")
	if err != nil {
		return nil, err
	}

	animDescs, err := internal.ReadSlice[AnimDesc](buf, int64(header.LocalAnimationOffset), int64(header.LocalAnimationCount), "animation descriptions")
	if err != nil {
		return nil, err
	}
	animDescNames, err := readNames(buf, header.LocalAnimationOffset, animDescs, func(a AnimDesc) int32 { return a.NameIndex }, "animation")
	if err != nil {
		return nil, err
	}

	sequenceDescs, err := internal.ReadSlice[SequenceDesc](buf, int64(header.LocalSequenceOffset), int64(header.LocalSequenceCount), "sequence descriptions")
	if err != nil {
		return nil, err
	}
	sequenceLabels, err := readNames(buf, header.LocalSequenceOffset, sequenceDescs, func(s SequenceDesc) int32 { return s.LabelIndex }, "sequence")
	if err != nil {
		return nil, err
	}
	sequenceActivityNames, err := readNames(buf, header.LocalSequenceOffset, sequenceDescs, func(s SequenceDesc) int32 { return s.ActivityNameIndex }, "sequence activity")
	if err != nil {
		return nil, err
	}

	textures, err := internal.ReadSlice[Texture](buf, int64(header.TextureOffset), int64(header.TextureCount), "textures")
	if err != nil {
		return nil, err
	}

	textureNames, err := readNames(buf, header.TextureOffset, textures, func(t Texture) int32 { return t.NameIndex }, "texture")
	if err != nil {
		return nil, err
	}

	textureDirOffsets, err := internal.ReadSlice[int32](buf, int64(header.TextureDirOffset), int64(header.TextureDirCount), "texture directory offsets")
	if err != nil {
		return nil, err
	}

	textureDirs := make([]string, len(textureDirOffsets))
	for i, offset := range textureDirOffsets {
		path, err := internal.CString(buf, int64(offset), 256, fmt.Sprintf("texture directory %d", i))
		if err != nil {
			return nil, err
		}
		textureDirs[i] = path
	}

	skinFamilies, err := reader.readSkinFamilies(buf, header)
	if err != nil {
		return nil, err
	}

	surfaceProp, keyValues, err := reader.readHeaderStrings(buf, header)
	if err != nil {
		return nil, err
	}

	// Parse body parts hierarchy (body parts → models → meshes)
	bodyParts, err := reader.readBodyParts(buf, header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body parts: %w", err)
	}

	return &Mdl{
		Header:                *header,
		Name:                  fixedString(header.Name[:]),
		SurfaceProp:           surfaceProp,
		KeyValues:             keyValues,
		Bones:                 bones,
		BoneNames:             boneNames,
		BoneSurfaceProps:      boneSurfaceProps,
		BoneControllers:       boneControllers,
		HitboxSet:             hitboxSets,
		HitboxSetNames:        hitboxSetNames,
		AnimDescs:             animDescs,
		AnimDescNames:         animDescNames,
		SequenceDescs:         sequenceDescs,
		SequenceLabels:        sequenceLabels,
		SequenceActivityNames: sequenceActivityNames,
		Textures:              textures,
		TextureNames:          textureNames,
		TextureDirs:           textureDirs,
		SkinFamilies:          skinFamilies,
		BodyParts:             bodyParts,
	}, nil
}

// readNames reads a name for each of items, which are stored back to back from start. Each name's offset, from
// nameIndex, is relative to its item; an offset of 0 means no name.
func readNames[T any](buf []byte, start int32, items []T, nameIndex func(T) int32, what string) ([]string, error) {
	var zero T
	size := int64(binary.Size(zero))
	names := make([]string, len(items))
	for i, item := range items {
		index := nameIndex(item)
		if index == 0 {
			continue
		}
		name, err := internal.CString(buf, int64(start)+int64(i)*size+int64(index), 256, fmt.Sprintf("%s %d name", what, i))
		if err != nil {
			return nil, err
		}
		names[i] = name
	}
	return names, nil
}

// fixedString returns the text of a null-padded fixed-size string
func fixedString(b []byte) string {
	if end := bytes.IndexByte(b, 0); end >= 0 {
		return string(b[:end])
	}
	return string(b)
}

// readSkinFamilies reads the skin table: for each skin, the texture index for each mesh material
func (reader *Reader) readSkinFamilies(buf []byte, header *Studiohdr) ([][]int16, error) {
	numSkinRefs, numSkinFamilies := int64(header.SkinReferenceCount), int64(header.SkinReferenceFamilyCount)
	if numSkinRefs < 0 || numSkinFamilies < 0 {
		return nil, fmt.Errorf("invalid skin table size: %d skins of %d materials", numSkinFamilies, numSkinRefs)
	}
	if numSkinRefs == 0 {
		return nil, nil
	}
	skinRefs, err := internal.ReadSlice[int16](buf, int64(header.SkinReferenceIndex), numSkinFamilies*numSkinRefs, "skin families")
	if err != nil {
		return nil, err
	}
	return slices.Collect(slices.Chunk(skinRefs, int(numSkinRefs))), nil
}

// readHeaderStrings reads the model's surface property and key-values, whose offsets are relative to the header
func (reader *Reader) readHeaderStrings(buf []byte, header *Studiohdr) (surfaceProp string, keyValues string, err error) {
	if header.SurfacePropertyIndex != 0 {
		if surfaceProp, err = internal.CString(buf, int64(header.SurfacePropertyIndex), 256, "surface prop"); err != nil {
			return "", "", err
		}
	}
	if header.KeyValueCount > 0 {
		start, size := int64(header.KeyValueIndex), int64(header.KeyValueCount)
		if err := internal.CheckRange(buf, start, size, "key values"); err != nil {
			return "", "", err
		}
		keyValues = strings.TrimRight(string(buf[start:start+size]), "\x00")
	}
	return surfaceProp, keyValues, nil
}

// readHeader Reads studiohdr header information
func (reader *Reader) readHeader(buf []byte) (*Studiohdr, error) {
	header, err := internal.ReadStruct[Studiohdr](buf, 0, "header")
	return &header, err
}

// NewReader returns a new Reader.
func NewReader() *Reader {
	return new(Reader)
}

// readBodyParts parses all body parts, and the models and meshes within them
func (reader *Reader) readBodyParts(buf []byte, header *Studiohdr) ([]BodyPartData, error) {
	bodyPartHeaders, err := internal.ReadSlice[BodyPart](buf, int64(header.BodypartOffset), int64(header.BodyPartCount), "body parts")
	if err != nil {
		return nil, err
	}

	bodyPartNames, err := readNames(buf, header.BodypartOffset, bodyPartHeaders, func(b BodyPart) int32 { return b.NameIndex }, "body part")
	if err != nil {
		return nil, err
	}

	bodyPartSize := int64(binary.Size(BodyPart{}))
	modelSize := int64(binary.Size(Model{}))

	bodyParts := make([]BodyPartData, len(bodyPartHeaders))
	for i, bodyPart := range bodyPartHeaders {
		// ModelIndex is relative to the start of this body part
		modelStart := int64(header.BodypartOffset) + int64(i)*bodyPartSize + int64(bodyPart.ModelIndex)
		models, err := internal.ReadSlice[Model](buf, modelStart, int64(bodyPart.NumModels), "models")
		if err != nil {
			return nil, fmt.Errorf("failed to parse models for body part %d: %w", i, err)
		}

		modelData := make([]ModelData, len(models))
		for j, model := range models {
			// MeshIndex is relative to the start of this model
			meshStart := modelStart + int64(j)*modelSize + int64(model.MeshIndex)
			meshes, err := internal.ReadSlice[Mesh](buf, meshStart, int64(model.NumMeshes), "meshes")
			if err != nil {
				return nil, fmt.Errorf("failed to parse meshes for model %d in body part %d: %w", j, i, err)
			}

			modelData[j] = ModelData{
				Header: model,
				Name:   fixedString(model.Name[:]),
				Meshes: meshes,
			}
		}

		bodyParts[i] = BodyPartData{
			Header: bodyPart,
			Name:   bodyPartNames[i],
			Models: modelData,
		}
	}

	return bodyParts, nil
}
