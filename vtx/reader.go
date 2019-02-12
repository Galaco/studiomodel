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
	err := reader.getByteBuffer()
	if err != nil {
		return nil, err
	}

	// Read header
	header, err := reader.readHeader()
	if err != nil {
		return nil, err
	}

	out := Vtx{}

	stream := seeker{
		buf:      &reader.buf,
		Position: 0,
		Begin:    0,
	}

	stream.Seek(header.BodyPartOffset, stream.Begin)
	func() {
		size := sizeOf(&bodyPartHeader{})
		start := stream.Position

		bodyParts, e := reader.readBodyParts(start, header.NumBodyParts)
		err = e

		//bodyparts
		for i, bodyPart := range bodyParts {
			bodyPartOut := BodyPart{}
			stream.Seek(start+(int32(i)*size), stream.Begin)

			stream.Seek(bodyPart.ModelOffset, stream.Position)
			//callback
			func() {
				start := stream.Position
				size := sizeOf(&modelHeader{})

				models, e := reader.readModels(start, bodyPart.NumModels)
				err = e
				for j, model := range models {
					modelOut := Model{}
					stream.Seek(start+(int32(j)*size), stream.Begin)

					stream.Seek(model.LODOffset, stream.Position)
					//callback
					func() {
						start := stream.Position
						size := sizeOf(&modelLODHeader{})

						modelLods, e := reader.readModelLODs(start, model.NumLODs)
						err = e
						for k, modelLod := range modelLods {
							modelLODOut := ModelLOD{}
							stream.Seek(start+(int32(k)*size), stream.Begin)

							stream.Seek(modelLod.MeshOffset, stream.Position)
							//callback
							func() {
								start := stream.Position
								size := int32(9) //sizeOf(&meshHeader{}) //vtx ignores trailing byte 4-byte alignment

								meshes, e := reader.readMeshes(start, modelLod.NumMeshes)
								err = e
								for l, mesh := range meshes {
									meshOut := Mesh{}
									stream.Seek(start+(int32(l)*size), stream.Begin)

									stream.Seek(mesh.StripGroupHeaderOffset, stream.Position)
									//callback
									func() {
										start := stream.Position
										size := int32(25) //sizeOf(&stripGroupHeader{}) //vtx ignores trailing byte 4-byte alignment

										stripGroups, e := reader.readStripGroups(start, mesh.NumStripGroups)
										err = e
										for m, stripGroup := range stripGroups {
											stripGroupOut := StripGroup{}
											stream.Seek(start+(int32(m)*size), stream.Begin)

											//callback
											func() {
												start := stream.Position
												size := int32(15) //sizeOf(&Strip{})

												stripGroupOut.Vertexes, e = reader.readVertices(start+stripGroup.VertOffset, stripGroup.NumVerts)
												err = e
												stripGroupOut.Indices, e = reader.readIndices(start+stripGroup.IndexOffset, stripGroup.NumIndices)
												err = e
												stripGroupOut.Strips, e = reader.readStrips(start+stripGroup.StripOffset, stripGroup.NumStrips)
												err = e

												for n := range stripGroupOut.Strips {
													stream.Seek(start+(int32(n)*size), stream.Begin)
												}
											}()
											meshOut.StripGroups = append(meshOut.StripGroups, stripGroupOut)
										}
									}()
									modelLODOut.Meshes = append(modelLODOut.Meshes, meshOut)
								}
							}()
							modelOut.LODS = append(modelOut.LODS, modelLODOut)
						}
					}()
					bodyPartOut.Models = append(bodyPartOut.Models, modelOut)
				}
			}()
			out.BodyParts = append(out.BodyParts, bodyPartOut)
		}
	}()

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
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]bodyPartHeader, 0), errors.New("body part data out of bounds")
	}
	ret := make([]bodyPartHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readModels(offset int32, num int32) ([]modelHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelHeader, 0), errors.New("model data out of bounds")
	}
	ret := make([]modelHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readModelLODs(offset int32, num int32) ([]modelLODHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelLODHeader, 0), errors.New("model lod data out of bounds")
	}
	ret := make([]modelLODHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readMeshes(offset int32, num int32) ([]meshHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]meshHeader, 0), errors.New("mesh data out of bounds")
	}
	ret := make([]meshHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readStripGroups(offset int32, num int32) ([]stripGroupHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]stripGroupHeader, 0), errors.New("strip group data out of bounds")
	}
	ret := make([]stripGroupHeader, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readIndices(offset int32, num int32) ([]uint16, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]uint16, 0), errors.New("indices data out of bounds")
	}
	ret := make([]uint16, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readVertices(offset int32, num int32) ([]Vertex, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Vertex, 0), errors.New("vertex data out of bounds")
	}
	ret := make([]Vertex, num)
	binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	return ret, nil
}

func (reader *Reader) readStrips(offset int32, num int32) ([]Strip, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Strip, 0), errors.New("strip data out of bounds")
	}
	ret := make([]Strip, num)
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
	buf      *[]byte
	Position int32
	Begin    int32
}

func (s *seeker) Seek(offset int32, start int32) {
	s.Position = start + offset
}

func (s *seeker) Read(num int32, size int32, callback func([]byte)) {
	s.Position += (num * size)
	callback((*s.buf)[s.Position-(num*size) : s.Position])
}

func sizeOf(t interface{}) int32 {
	typeName := reflect.TypeOf(t)
	return int32(typeName.Elem().Size())
	//return int32(unsafe.Sizeof(t))
}

func isPropertyValid(offset int32, num int32, bufferSize int) bool {
	if num < 1 || offset < 1 {
		return false
	}
	return true
}
