package mdl

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"unsafe"
)

type Reader struct {
	stream io.Reader
	buf    []byte
}

func (reader *Reader) Read() (*Mdl, error) {
	err := reader.getByteBuffer()
	if err != nil {
		return nil, err
	}

	header, err := reader.readHeader()
	if err != nil {
		return nil, err
	}
	if header.StudioHDR2Index > 0 {
		// reader header2
	}

	// Read all properties
	bones := make([]Bone, header.BoneCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.BoneOffset:header.BoneOffset + int32(int(unsafe.Sizeof(Bone{}))*len(bones))]), binary.LittleEndian, &bones)

	boneControllers := make([]BoneController, header.BoneControllerCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.BoneControllerOffset:header.BoneControllerOffset + int32(int(unsafe.Sizeof(BoneController{}))*len(boneControllers))]), binary.LittleEndian, &boneControllers)

	hitboxSets := make([]HitboxSet, header.HitboxCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.HitboxOffset:header.HitboxOffset + int32(int(unsafe.Sizeof(HitboxSet{}))*len(hitboxSets))]), binary.LittleEndian, &hitboxSets)

	animDescs := make([]AnimDesc, header.LocalAnimationCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.LocalAnimationOffset:header.LocalAnimationOffset + int32(int(unsafe.Sizeof(AnimDesc{}))*len(animDescs))]), binary.LittleEndian, &animDescs)

	sequenceDescs := make([]SequenceDesc, header.LocalSequenceCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.LocalSequenceOffset:header.LocalSequenceOffset + int32(int(unsafe.Sizeof(SequenceDesc{}))*len(sequenceDescs))]), binary.LittleEndian, &sequenceDescs)

	textures := make([]Texture, header.TextureCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.TextureOffset:header.TextureOffset + int32(int(unsafe.Sizeof(Texture{}))*len(textures))]), binary.LittleEndian, &textures)

	textureNames := make([]string, len(textures))
	for _,tex := range textures {
		s := make([]byte, 255)
		binary.Read(bytes.NewBuffer(reader.buf[header.TextureOffset + tex.NameIndex:header.TextureOffset + tex.NameIndex + 255]), binary.LittleEndian, &s)
		name := strings.Split(string(s), "\x00")

		for j := 0; j < len(textureNames); j++ {
			textureNames[j] = name[j]
		}
		break
	}

	textureDirOffsets := make([]int32, header.TextureDirCount)
	binary.Read(bytes.NewBuffer(reader.buf[header.TextureDirOffset:header.TextureDirOffset + int32(int(unsafe.Sizeof(int32(0)))*len(textureDirOffsets))]), binary.LittleEndian, &textureDirOffsets)

	textureDirs := make([]string, header.TextureDirCount)
	for i,offset := range textureDirOffsets {
		s := make([]byte, 255)
		binary.Read(bytes.NewBuffer(reader.buf[offset:offset + 255]), binary.LittleEndian, &s)
		paths := strings.Split(string(s), "\x00")
		textureDirs[i] = paths[0]
	}



	return &Mdl{
		Header: *header,
		Bones: bones,
		BoneControllers: boneControllers,
		HitboxSet: hitboxSets,
		AnimDescs: animDescs,
		SequenceDescs: sequenceDescs,
		Textures: textures,
		TextureNames: textureNames,
		TextureDirs: textureDirs,
	}, nil
}

// Reads studiohdr header information
func (reader *Reader) readHeader() (*Studiohdr, error) {
	header := Studiohdr{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(reader.buf[:headerSize]), binary.LittleEndian, &header)

	return &header, err
}

// Read stream to []byte buffer
func (reader *Reader) getByteBuffer() error {
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(reader.stream)
	if err == nil {
		reader.buf = buf.Bytes()
	}

	return err
}
