package vtx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"unsafe"
)

type Reader struct {
	stream io.Reader
	buf    []byte
}

func (reader *Reader) Read() (*Vtx, error) {
	var err error
	err = reader.getByteBuffer()
	if err != nil {
		return nil, err
	}

	// Read header
	header, err := reader.readHeader()
	if err != nil {
		return nil, err
	}

	out := Vtx{}

	s := seeker{
		buf:     &reader.buf,
		Current: 0,
		Begin:   0,
	}

	s.Seek(header.BodyPartOffset, s.Begin)
	func(stream *seeker) {
		start := stream.Current

		bodyParts, e := reader.readBodyParts(start, header.NumBodyParts)
		err = e

		for i, bodyPart := range bodyParts {
			bodyPartOut := BodyPart{}
			stream.Seek(start+(int32(i)*sizeOf(&bodyPart))+bodyPart.ModelOffset, stream.Begin)
			func(stream *seeker) {
				start := stream.Current

				models, e := reader.readModels(start, bodyPart.NumModels)
				err = e
				for j, model := range models {
					modelOut := Model{}
					stream.Seek(start+(int32(j)*sizeOf(&model))+model.LODOffset, stream.Begin)
					//modelLODS
					func(stream *seeker) {
						start := stream.Current

						modelLods, e := reader.readModelLODs(start, model.NumLODs)
						err = e
						for k, modelLod := range modelLods {
							modelLODOut := ModelLOD{}
							stream.Seek(start+(int32(k)*sizeOf(&modelLod))+modelLod.MeshOffset, stream.Begin)
							// Meshes
							func(stream *seeker) {
								start := stream.Current

								meshes, e := reader.readMeshes(start, modelLod.NumMeshes)
								err = e
								for l, mesh := range meshes {
									meshOut := Mesh{}
									stream.Seek(start+(int32(l)*sizeOf(&mesh))+mesh.StripGroupHeaderOffset, stream.Begin)
									// StripGroups
									func(stream *seeker) {
										start := stream.Current

										stripGroups, e := reader.readStripGroups(start, mesh.NumStripGroups)
										err = e
										for m, stripGroup := range stripGroups {
											stripGroupOut := StripGroup{}
											stream.Seek(start+(int32(m)*sizeOf(&stripGroup)), stream.Begin)

											// Verts,indices,strips
											func(stream *seeker) {
												start := stream.Current

												stripGroupOut.Vertexes, e = reader.readVertices(start+stripGroup.VertOffset, stripGroup.NumVerts)

												err = e
												stripGroupOut.Indices, e = reader.readIndices(start+stripGroup.IndexOffset, stripGroup.NumIndices)

												err = e
												stripGroupOut.Strips, e = reader.readStrips(start+stripGroup.StripOffset, stripGroup.NumStrips)
												err = e
											}(&s)
											meshOut.StripGroups = append(meshOut.StripGroups, stripGroupOut)
										}
									}(&s)
									modelLODOut.Meshes = append(modelLODOut.Meshes, meshOut)
								}
							}(&s)
							modelOut.LODS = append(modelOut.LODS, modelLODOut)
						}
					}(&s)
					bodyPartOut.Models = append(bodyPartOut.Models, modelOut)
				}
			}(&s)
			out.BodyParts = append(out.BodyParts, bodyPartOut)
		}
	}(&s)

	if err != nil {
		return nil, err
	}

	return &out, nil
}

// Reads studiohdr header information
func (reader *Reader) readHeader() (header, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(reader.buf[:headerSize]), binary.LittleEndian, &header)

	return header, err
}

func (reader *Reader) readBodyParts(offset int32, num int32) ([]bodyPartHeader, error) {
	ret := make([]bodyPartHeader, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("body part data out of bounds")
	}

	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readModels(offset int32, num int32) ([]modelHeader, error) {
	ret := make([]modelHeader, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("model data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readModelLODs(offset int32, num int32) ([]modelLODHeader, error) {
	ret := make([]modelLODHeader, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("model lod data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readMeshes(offset int32, num int32) ([]meshHeader, error) {
	ret := make([]meshHeader, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("mesh data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readStripGroups(offset int32, num int32) ([]stripGroupHeader, error) {
	ret := make([]stripGroupHeader, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("strip group data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readIndices(offset int32, num int32) ([]uint16, error) {
	ret := make([]uint16, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("indices data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readVertices(offset int32, num int32) ([]Vertex, error) {
	ret := make([]Vertex, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("vertex data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readStrips(offset int32, num int32) ([]Strip, error) {
	ret := make([]Strip, num)
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return ret, errors.New("strip data out of bounds")
	}
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
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

type seeker struct {
	buf     *[]byte
	Current int32
	Begin   int32
}

func (s *seeker) Seek(offset int32, start int32) {
	s.Current = start + offset
}

func (s *seeker) Read(num int32, size int32, callback func([]byte)) {
	s.Current += (num * size)
	callback((*s.buf)[s.Current-(num*size) : s.Current])
}

func sizeOf(t interface{}) int32 {
	typeName := reflect.TypeOf(t)
	return int32(typeName.Elem().Size())
	//return int32(unsafe.Sizeof(t))
}

func isPropertyValid(offset int32, num int32, bufferSize int) bool {
	if int64(int64(offset)*int64(num)) > int64(bufferSize) {
		return false
	}
	return true
}
