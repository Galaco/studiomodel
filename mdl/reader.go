package mdl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
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
	var buf []byte
	byteBuf := bytes.Buffer{}
	_, err := byteBuf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	buf = byteBuf.Bytes()

	// Validate minimum file size
	if len(buf) < int(unsafe.Sizeof(Studiohdr{})) {
		return nil, fmt.Errorf("mdl file too small: %d bytes, expected at least %d", len(buf), unsafe.Sizeof(Studiohdr{}))
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

	// Validate counts are non-negative
	if header.BoneCount < 0 || header.BoneControllerCount < 0 || header.HitboxCount < 0 ||
		header.LocalAnimationCount < 0 || header.LocalSequenceCount < 0 || header.TextureCount < 0 || header.TextureDirCount < 0 {
		return nil, fmt.Errorf("MDL header contains negative counts")
	}

	// Validate offsets are within file bounds
	if header.DataLength > 0 && int(header.DataLength) != len(buf) {
		return nil, fmt.Errorf("MDL data length mismatch: header says %d, file is %d bytes", header.DataLength, len(buf))
	}

	//if header.StudioHDR2Index > 0 {
	//	// TODO: reader header2
	//}

	// Read all properties with bounds checking
	bones := make([]Bone, header.BoneCount)
	if header.BoneCount > 0 {
		boneSize := int32(int(unsafe.Sizeof(Bone{})) * len(bones))
		if err := validateOffset(buf, header.BoneOffset, boneSize, "bones"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.BoneOffset:header.BoneOffset+boneSize]), binary.LittleEndian, &bones)
		if err != nil {
			return nil, fmt.Errorf("failed to read bones at offset %d: %w", header.BoneOffset, err)
		}
	}

	boneControllers := make([]BoneController, header.BoneControllerCount)
	if header.BoneControllerCount > 0 {
		boneControllerSize := int32(int(unsafe.Sizeof(BoneController{})) * len(boneControllers))
		if err := validateOffset(buf, header.BoneControllerOffset, boneControllerSize, "bone controllers"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.BoneControllerOffset:header.BoneControllerOffset+boneControllerSize]), binary.LittleEndian, &boneControllers)
		if err != nil {
			return nil, fmt.Errorf("failed to read bone controllers at offset %d: %w", header.BoneControllerOffset, err)
		}
	}

	hitboxSets := make([]HitboxSet, header.HitboxCount)
	if header.HitboxCount > 0 {
		hitboxSetSize := int32(int(unsafe.Sizeof(HitboxSet{})) * len(hitboxSets))
		if err := validateOffset(buf, header.HitboxOffset, hitboxSetSize, "hitbox sets"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.HitboxOffset:header.HitboxOffset+hitboxSetSize]), binary.LittleEndian, &hitboxSets)
		if err != nil {
			return nil, fmt.Errorf("failed to read hitbox sets at offset %d: %w", header.HitboxOffset, err)
		}
	}

	animDescs := make([]AnimDesc, header.LocalAnimationCount)
	if header.LocalAnimationCount > 0 {
		animDescSize := int32(int(unsafe.Sizeof(AnimDesc{})) * len(animDescs))
		if err := validateOffset(buf, header.LocalAnimationOffset, animDescSize, "animation descriptions"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.LocalAnimationOffset:header.LocalAnimationOffset+animDescSize]), binary.LittleEndian, &animDescs)
		if err != nil {
			return nil, fmt.Errorf("failed to read animation descriptions at offset %d: %w", header.LocalAnimationOffset, err)
		}
	}

	sequenceDescs := make([]SequenceDesc, header.LocalSequenceCount)
	if header.LocalSequenceCount > 0 {
		sequenceDescSize := int32(int(unsafe.Sizeof(SequenceDesc{})) * len(sequenceDescs))
		if err := validateOffset(buf, header.LocalSequenceOffset, sequenceDescSize, "sequence descriptions"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.LocalSequenceOffset:header.LocalSequenceOffset+sequenceDescSize]), binary.LittleEndian, &sequenceDescs)
		if err != nil {
			return nil, fmt.Errorf("failed to read sequence descriptions at offset %d: %w", header.LocalSequenceOffset, err)
		}
	}

	textures := make([]Texture, header.TextureCount)
	if header.TextureCount > 0 {
		textureSize := int32(int(unsafe.Sizeof(Texture{})) * len(textures))
		if err := validateOffset(buf, header.TextureOffset, textureSize, "textures"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.TextureOffset:header.TextureOffset+textureSize]), binary.LittleEndian, &textures)
		if err != nil {
			return nil, fmt.Errorf("failed to read textures at offset %d: %w", header.TextureOffset, err)
		}
	}

	textureNames := make([]string, header.TextureCount)
	for i := range textures {
		nameOffset := header.TextureOffset + int32(int(unsafe.Sizeof(Texture{}))*i) + textures[i].NameIndex
		if nameOffset < 0 || int(nameOffset) >= len(buf) {
			return nil, fmt.Errorf("texture %d name offset %d out of bounds", i, nameOffset)
		}
		name, err := readCString(buf, int(nameOffset), 256)
		if err != nil {
			return nil, fmt.Errorf("failed to read texture %d name at offset %d: %w", i, nameOffset, err)
		}
		textureNames[i] = name
	}

	textureDirOffsets := make([]int32, header.TextureDirCount)
	if header.TextureDirCount > 0 {
		textureDirOffsetsSize := int32(int(unsafe.Sizeof(int32(0))) * len(textureDirOffsets))
		if err := validateOffset(buf, header.TextureDirOffset, textureDirOffsetsSize, "texture directory offsets"); err != nil {
			return nil, err
		}
		err = binary.Read(bytes.NewBuffer(buf[header.TextureDirOffset:header.TextureDirOffset+textureDirOffsetsSize]), binary.LittleEndian, &textureDirOffsets)
		if err != nil {
			return nil, fmt.Errorf("failed to read texture directory offsets at %d: %w", header.TextureDirOffset, err)
		}
	}

	textureDirs := make([]string, header.TextureDirCount)
	for i, offset := range textureDirOffsets {
		if offset < 0 || int(offset) >= len(buf) {
			return nil, fmt.Errorf("texture directory %d offset %d out of bounds", i, offset)
		}
		path, err := readCString(buf, int(offset), 256)
		if err != nil {
			return nil, fmt.Errorf("failed to read texture directory %d at offset %d: %w", i, offset, err)
		}
		textureDirs[i] = path
	}

	// Parse body parts hierarchy (body parts → models → meshes)
	var bodyParts []BodyPartData
	if header.BodyPartCount > 0 {
		bodyPartHeaders, err := reader.readBodyParts(buf, header)
		if err != nil {
			return nil, fmt.Errorf("failed to parse body parts: %w", err)
		}

		// Parse models and meshes for each body part
		bodyParts = make([]BodyPartData, len(bodyPartHeaders))
		for i, bodyPartHeader := range bodyPartHeaders {
			bodyPartOffset := header.BodypartOffset + int32(i)*int32(unsafe.Sizeof(BodyPart{}))

			// Read models for this body part
			models, err := reader.readModelsForBodyPart(buf, &bodyPartHeader, bodyPartOffset)
			if err != nil {
				return nil, fmt.Errorf("failed to parse models for body part %d: %w", i, err)
			}

			// Parse meshes for each model
			modelData := make([]ModelData, len(models))
			for j, model := range models {
				modelOffset := bodyPartOffset + bodyPartHeader.ModelIndex + int32(j)*int32(unsafe.Sizeof(Model{}))

				meshes, err := reader.readMeshesForModel(buf, &model, modelOffset)
				if err != nil {
					return nil, fmt.Errorf("failed to parse meshes for model %d in body part %d: %w", j, i, err)
				}

				modelData[j] = ModelData{
					Header: model,
					Meshes: meshes,
				}
			}

			bodyParts[i] = BodyPartData{
				Header: bodyPartHeader,
				Models: modelData,
			}
		}
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
	header := Studiohdr{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(buf[:headerSize]), binary.LittleEndian, &header)

	return &header, err
}

// NewReader returns a new Reader.
func NewReader() *Reader {
	return new(Reader)
}

// validateOffset checks if the given offset and size are within buffer bounds
func validateOffset(buf []byte, offset int32, size int32, name string) error {
	if offset < 0 {
		return fmt.Errorf("%s offset is negative: %d", name, offset)
	}
	if size < 0 {
		return fmt.Errorf("%s size is negative: %d", name, size)
	}
	requiredSize := int(offset) + int(size)
	if requiredSize > len(buf) {
		return fmt.Errorf("%s data exceeds buffer: offset=%d size=%d required=%d bufferSize=%d", name, offset, size, requiredSize, len(buf))
	}
	return nil
}

// readCString reads a null-terminated string from the buffer with bounds checking
func readCString(buf []byte, offset int, maxLen int) (string, error) {
	if offset < 0 || offset >= len(buf) {
		return "", fmt.Errorf("string offset %d out of bounds (buffer size %d)", offset, len(buf))
	}

	end := offset
	limit := offset + maxLen
	if limit > len(buf) {
		limit = len(buf)
	}

	for end < limit && buf[end] != 0 {
		end++
	}

	return string(buf[offset:end]), nil
}

// readBodyParts parses all body part structures from the MDL file
func (reader *Reader) readBodyParts(buf []byte, header *Studiohdr) ([]BodyPart, error) {
	if header.BodyPartCount == 0 {
		return nil, nil
	}

	bodyParts := make([]BodyPart, header.BodyPartCount)
	bodyPartSize := int32(unsafe.Sizeof(BodyPart{}))
	totalSize := bodyPartSize * header.BodyPartCount

	if err := validateOffset(buf, header.BodypartOffset, totalSize, "body parts"); err != nil {
		return nil, err
	}

	err := binary.Read(
		bytes.NewBuffer(buf[header.BodypartOffset:header.BodypartOffset+totalSize]),
		binary.LittleEndian,
		&bodyParts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read body parts at offset %d: %w", header.BodypartOffset, err)
	}

	return bodyParts, nil
}

// readModelsForBodyPart parses all models within a body part
func (reader *Reader) readModelsForBodyPart(buf []byte, bodyPart *BodyPart, bodyPartOffset int32) ([]Model, error) {
	if bodyPart.NumModels == 0 {
		return nil, nil
	}

	models := make([]Model, bodyPart.NumModels)
	modelSize := int32(unsafe.Sizeof(Model{}))
	totalSize := modelSize * bodyPart.NumModels

	// ModelIndex is relative to the bodypart offset
	modelOffset := bodyPartOffset + bodyPart.ModelIndex

	fmt.Printf("[MDL DEBUG] readModelsForBodyPart:\n")
	fmt.Printf("[MDL DEBUG]   bodyPartOffset: %d (0x%X)\n", bodyPartOffset, bodyPartOffset)
	fmt.Printf("[MDL DEBUG]   bodyPart.NumModels: %d\n", bodyPart.NumModels)
	fmt.Printf("[MDL DEBUG]   bodyPart.ModelIndex: %d (0x%X)\n", bodyPart.ModelIndex, bodyPart.ModelIndex)
	fmt.Printf("[MDL DEBUG]   sizeof(Model{}): %d\n", unsafe.Sizeof(Model{}))
	fmt.Printf("[MDL DEBUG]   calculated modelOffset: %d (0x%X)\n", modelOffset, modelOffset)

	if err := validateOffset(buf, modelOffset, totalSize, "models"); err != nil {
		return nil, err
	}

	err := binary.Read(
		bytes.NewBuffer(buf[modelOffset:modelOffset+totalSize]),
		binary.LittleEndian,
		&models,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read models at offset %d: %w", modelOffset, err)
	}

	// Log what was actually read for each model
	for i, model := range models {
		fmt.Printf("[MDL DEBUG]   Model %d:\n", i)
		fmt.Printf("[MDL DEBUG]     Name: %s\n", string(model.Name[:]))
		fmt.Printf("[MDL DEBUG]     NumMeshes: %d (0x%X)\n", model.NumMeshes, model.NumMeshes)
		fmt.Printf("[MDL DEBUG]     MeshIndex: %d (0x%X)\n", model.MeshIndex, model.MeshIndex)
		fmt.Printf("[MDL DEBUG]     NumVertices: %d\n", model.NumVertices)
		fmt.Printf("[MDL DEBUG]     VertexIndex: %d (0x%X)\n", model.VertexIndex, model.VertexIndex)
		fmt.Printf("[MDL DEBUG]     Unknown fields (bytes 112-148):\n")
		fmt.Printf("[MDL DEBUG]       Unknown1: %d (0x%08X)\n", model.Unknown1, model.Unknown1)
		fmt.Printf("[MDL DEBUG]       Unknown2: %d (0x%08X)\n", model.Unknown2, model.Unknown2)
		fmt.Printf("[MDL DEBUG]       Unknown3: %d (0x%08X)\n", model.Unknown3, model.Unknown3)
		fmt.Printf("[MDL DEBUG]       Unknown4: %d (0x%08X)\n", model.Unknown4, model.Unknown4)
		fmt.Printf("[MDL DEBUG]       Unknown5: %d (0x%08X)\n", model.Unknown5, model.Unknown5)
		fmt.Printf("[MDL DEBUG]       Unknown6: %d (0x%08X)\n", model.Unknown6, model.Unknown6)
		fmt.Printf("[MDL DEBUG]       Unknown7: %d (0x%08X)\n", model.Unknown7, model.Unknown7)
		fmt.Printf("[MDL DEBUG]       Unknown8: %d (0x%08X)\n", model.Unknown8, model.Unknown8)
		fmt.Printf("[MDL DEBUG]       Unknown9: %d (0x%08X)\n", model.Unknown9, model.Unknown9)
	}

	return models, nil
}

// readMeshesForModel parses all meshes within a model
func (reader *Reader) readMeshesForModel(buf []byte, model *Model, modelOffset int32) ([]Mesh, error) {
	if model.NumMeshes == 0 {
		return nil, nil
	}

	// Debug logging to diagnose negative size issue
	fmt.Printf("[MDL DEBUG] readMeshesForModel called:\n")
	fmt.Printf("[MDL DEBUG]   modelOffset: %d (0x%X)\n", modelOffset, modelOffset)
	fmt.Printf("[MDL DEBUG]   model.NumMeshes: %d (0x%X)\n", model.NumMeshes, model.NumMeshes)
	fmt.Printf("[MDL DEBUG]   model.MeshIndex: %d (0x%X)\n", model.MeshIndex, model.MeshIndex)
	fmt.Printf("[MDL DEBUG]   sizeof(Mesh{}): %d\n", unsafe.Sizeof(Mesh{}))

	meshes := make([]Mesh, model.NumMeshes)
	meshSize := int32(unsafe.Sizeof(Mesh{}))
	totalSize := meshSize * model.NumMeshes

	fmt.Printf("[MDL DEBUG]   meshSize: %d\n", meshSize)
	fmt.Printf("[MDL DEBUG]   totalSize (meshSize * NumMeshes): %d\n", totalSize)

	// MeshIndex is relative to the model offset
	meshOffset := modelOffset + model.MeshIndex

	fmt.Printf("[MDL DEBUG]   calculated meshOffset: %d (0x%X)\n", meshOffset, meshOffset)
	fmt.Printf("[MDL DEBUG]   buffer length: %d\n", len(buf))

	if err := validateOffset(buf, meshOffset, totalSize, "meshes"); err != nil {
		return nil, err
	}

	err := binary.Read(
		bytes.NewBuffer(buf[meshOffset:meshOffset+totalSize]),
		binary.LittleEndian,
		&meshes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read meshes at offset %d: %w", meshOffset, err)
	}

	return meshes, nil
}
