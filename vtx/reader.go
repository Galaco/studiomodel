package vtx

import (
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"
)

type Reader struct {
	stream io.Reader
	buf    []byte
}

func (reader *Reader) Read() (*Vtx, error) {
	err := reader.getByteBuffer()
	if err != nil {
		return nil, err
	}

	// Read header
	header, err := reader.readHeader()
	if err != nil {
		return nil, err
	}

	offset := int32(0)

	//bodyparts
	offset += header.BodyPartOffset
	bodyParts := reader.readBodyParts(offset, header.NumBodyParts)

	//models
	models := make([]modelHeader, 0)
	modelLods := make([]modelLODHeader, 0)
	meshes := make([]meshHeader, 0)
	stripGroups := make([]stripGroupHeader, 0)
	indices := make([]uint16, 0)
	vertices := make([]float32, 0)
	strips := make([]stripHeader, 0)

	for i, part := range bodyParts {
		start := offset + (int32(i)*int32(unsafe.Sizeof(part))) + part.ModelOffset
		models = append(models, reader.readModels(start, part.NumModels)...)

		//modellods
		for j, model := range models {
			start := start + (int32(j)*int32(unsafe.Sizeof(model))) + model.LODOffset
			modelLods = append(modelLods, reader.readModelLODs(start, model.NumLODs)...)

			//meshes
			for k, modelLod := range modelLods {
				start := start + (int32(k)*int32(unsafe.Sizeof(modelLod))) + modelLod.MeshOffset
				meshes = append(meshes, reader.readMeshes(start, modelLod.NumMeshes)...)

				//stripgroups
				for l, mesh := range meshes {
					start := start + (int32(l)*int32(unsafe.Sizeof(mesh))) + mesh.StripGroupHeaderOffset
					stripGroups = append(stripGroups, reader.readStripGroups(start, mesh.NumStripGroups)...)

					for m, stripGroup := range stripGroups {
						start := start + (int32(m)*int32(unsafe.Sizeof(stripGroup)))
						vertices = append(vertices, reader.readVertices(start + stripGroup.VertOffset, stripGroup.NumVerts)...)
						indices = append(indices, reader.readIndices(start + stripGroup.IndexOffset, stripGroup.NumIndices)...)
						strips = append(strips, reader.readStrips(start + stripGroup.StripOffset, stripGroup.NumStrips)...)
						//
						//for n, strip := range strips {
						//	start := start + (int32(n)*int32(unsafe.Sizeof(stripHeader{})))
						//
						//
						//}

					}
				}

			}
		}
	}

	////indices
	//indices := make([]uint16, 0)
	//for _, stripGroup := range stripGroups {
	//	indices = append(indices, reader.readIndices(offset+stripGroup.IndexOffset, stripGroup.NumIndices)...)
	//}
	//offset += int32(stripGroups[len(stripGroups)-1].IndexOffset)

	////vertices
	//vertices := make([]float32, 0)
	//for _, stripGroup := range stripGroups {
	//	vertices = append(vertices, reader.readVertices(offset+stripGroup.VertOffset, stripGroup.NumVerts)...)
	//}
	//offset += int32(stripGroups[len(stripGroups)-1].VertOffset)
	//
	////strips
	//strips := make([]stripHeader, 0)
	//for _, stripGroup := range stripGroups {
	//	strips = append(strips, reader.readStrips(offset+stripGroup.StripOffset, stripGroup.NumStrips)...)
	//}
	//offset += int32(stripGroups[len(stripGroups)-1].StripOffset)

	//vertexes

	return &Vtx{
		Header:    header,
		BodyParts: bodyParts,
		Models:    models,
	}, nil
}

// Reads studiohdr header information
func (reader *Reader) readHeader() (header, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(reader.buf[:headerSize]), binary.LittleEndian, &header)

	return header, err
}

func (reader *Reader) readBodyParts(offset int32, num int32) []bodyPartHeader {
	ret := make([]bodyPartHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readModels(offset int32, num int32) []modelHeader {
	ret := make([]modelHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readModelLODs(offset int32, num int32) []modelLODHeader {
	ret := make([]modelLODHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readMeshes(offset int32, num int32) []meshHeader {
	ret := make([]meshHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readStripGroups(offset int32, num int32) []stripGroupHeader {
	ret := make([]stripGroupHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readIndices(offset int32, num int32) []uint16 {
	ret := make([]uint16, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readVertices(offset int32, num int32) []float32 {
	ret := make([]float32, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
}

func (reader *Reader) readStrips(offset int32, num int32) []stripHeader {
	ret := make([]stripHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret
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
