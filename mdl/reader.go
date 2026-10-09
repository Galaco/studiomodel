package mdl

import (
	"encoding/binary"
	"fmt"
	"io"

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

	boneControllers, err := internal.ReadSlice[BoneController](buf, int64(header.BoneControllerOffset), int64(header.BoneControllerCount), "bone controllers")
	if err != nil {
		return nil, err
	}

	hitboxSets, err := internal.ReadSlice[HitboxSet](buf, int64(header.HitboxOffset), int64(header.HitboxCount), "hitbox sets")
	if err != nil {
		return nil, err
	}

	animDescs, err := internal.ReadSlice[AnimDesc](buf, int64(header.LocalAnimationOffset), int64(header.LocalAnimationCount), "animation descriptions")
	if err != nil {
		return nil, err
	}

	sequenceDescs, err := internal.ReadSlice[SequenceDesc](buf, int64(header.LocalSequenceOffset), int64(header.LocalSequenceCount), "sequence descriptions")
	if err != nil {
		return nil, err
	}

	textures, err := internal.ReadSlice[Texture](buf, int64(header.TextureOffset), int64(header.TextureCount), "textures")
	if err != nil {
		return nil, err
	}

	// Texture name offsets are relative to the start of each texture
	textureSize := int64(binary.Size(Texture{}))
	textureNames := make([]string, len(textures))
	for i := range textures {
		nameOffset := int64(header.TextureOffset) + textureSize*int64(i) + int64(textures[i].NameIndex)
		name, err := internal.CString(buf, nameOffset, 256, fmt.Sprintf("texture %d name", i))
		if err != nil {
			return nil, err
		}
		textureNames[i] = name
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

	// Parse body parts hierarchy (body parts → models → meshes)
	bodyParts, err := reader.readBodyParts(buf, header)
	if err != nil {
		return nil, fmt.Errorf("failed to parse body parts: %w", err)
	}

	return &Mdl{
		Header:          *header,
		Bones:           bones,
		BoneControllers: boneControllers,
		HitboxSet:       hitboxSets,
		AnimDescs:       animDescs,
		SequenceDescs:   sequenceDescs,
		Textures:        textures,
		TextureNames:    textureNames,
		TextureDirs:     textureDirs,
		BodyParts:       bodyParts,
	}, nil
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
				Meshes: meshes,
			}
		}

		bodyParts[i] = BodyPartData{
			Header: bodyPart,
			Models: modelData,
		}
	}

	return bodyParts, nil
}
