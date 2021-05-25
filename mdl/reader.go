package mdl

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"unsafe"
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

	header, err := reader.readHeader(buf)
	if err != nil {
		return nil, err
	}
	//if header.StudioHDR2Index > 0 {
	//	// reader header2
	//}

	// Read all properties
	bones := make([]Bone, header.BoneCount)
	err = binary.Read(bytes.NewBuffer(buf[header.BoneOffset:header.BoneOffset+int32(int(unsafe.Sizeof(Bone{}))*len(bones))]), binary.LittleEndian, &bones)
	if err != nil {
		return nil, err
	}

	boneControllers := make([]BoneController, header.BoneControllerCount)
	err = binary.Read(bytes.NewBuffer(buf[header.BoneControllerOffset:header.BoneControllerOffset+int32(int(unsafe.Sizeof(BoneController{}))*len(boneControllers))]), binary.LittleEndian, &boneControllers)
	if err != nil {
		return nil, err
	}

	hitboxSets := make([]HitboxSet, header.HitboxCount)
	err = binary.Read(bytes.NewBuffer(buf[header.HitboxOffset:header.HitboxOffset+int32(int(unsafe.Sizeof(HitboxSet{}))*len(hitboxSets))]), binary.LittleEndian, &hitboxSets)
	if err != nil {
		return nil, err
	}

	animDescs := make([]AnimDesc, header.LocalAnimationCount)
	err = binary.Read(bytes.NewBuffer(buf[header.LocalAnimationOffset:header.LocalAnimationOffset+int32(int(unsafe.Sizeof(AnimDesc{}))*len(animDescs))]), binary.LittleEndian, &animDescs)
	if err != nil {
		return nil, err
	}

	sequenceDescs := make([]SequenceDesc, header.LocalSequenceCount)
	err = binary.Read(bytes.NewBuffer(buf[header.LocalSequenceOffset:header.LocalSequenceOffset+int32(int(unsafe.Sizeof(SequenceDesc{}))*len(sequenceDescs))]), binary.LittleEndian, &sequenceDescs)
	if err != nil {
		return nil, err
	}

	textures := make([]Texture, header.TextureCount)
	err = binary.Read(bytes.NewBuffer(buf[header.TextureOffset:header.TextureOffset+int32(int(unsafe.Sizeof(Texture{}))*len(textures))]), binary.LittleEndian, &textures)
	if err != nil {
		return nil, err
	}

	textureNames := make([]string, header.TextureCount)
	for i := range textures {
		textureNames[i] = strings.SplitN(
			string(buf[header.TextureOffset+int32(int(unsafe.Sizeof(Texture{}))*i)+textures[i].NameIndex:]),
			"\x00",
			int(header.TextureCount+1))[0]
	}

	textureDirOffsets := make([]int32, header.TextureDirCount)
	err = binary.Read(bytes.NewBuffer(buf[header.TextureDirOffset:header.TextureDirOffset+int32(int(unsafe.Sizeof(int32(0)))*len(textureDirOffsets))]), binary.LittleEndian, &textureDirOffsets)
	if err != nil {
		return nil, err
	}

	textureDirs := make([]string, header.TextureDirCount)
	for i, offset := range textureDirOffsets {
		s := make([]byte, 255)
		err = binary.Read(bytes.NewBuffer(buf[offset:offset+255]), binary.LittleEndian, &s)
		if err != nil {
			return nil, err
		}
		paths := strings.Split(string(s), "\x00")
		textureDirs[i] = paths[0]
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
