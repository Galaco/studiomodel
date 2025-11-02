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
