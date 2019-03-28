package vtx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/galaco/StudioModel/internal"
	"io"
	"unsafe"
)

// Reader
type Reader struct {
	buf []byte
}

// Read parses a stream to a Vtx struct
func (reader *Reader) Read(stream io.Reader) (*Vtx, error) {
	byteBuf := bytes.Buffer{}
	_, err := byteBuf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	reader.buf = byteBuf.Bytes()

	// Read header
	header, err := reader.readHeader()
	if err != nil {
		return nil, err
	}

	out := Vtx{}

	streamInternal := internal.NewSeeker(&reader.buf)

	streamInternal.Seek(header.BodyPartOffset, streamInternal.Begin)
	func() {
		size := internal.SizeOf(&bodyPartHeader{})
		start := streamInternal.Position

		bodyParts, e := reader.readBodyParts(start, header.NumBodyParts)
		err = e

		//bodyparts
		for i, bodyPart := range bodyParts {
			bodyPartOut := BodyPart{}
			streamInternal.Seek(start+(int32(i)*size), streamInternal.Begin)

			streamInternal.Seek(bodyPart.ModelOffset, streamInternal.Position)
			//callback
			func() {
				start := streamInternal.Position
				size := internal.SizeOf(&modelHeader{})

				models, e := reader.readModels(start, bodyPart.NumModels)
				err = e
				for j, model := range models {
					modelOut := Model{}
					streamInternal.Seek(start+(int32(j)*size), streamInternal.Begin)

					streamInternal.Seek(model.LODOffset, streamInternal.Position)
					//callback
					func() {
						start := streamInternal.Position
						size := internal.SizeOf(&modelLODHeader{})

						modelLods, e := reader.readModelLODs(start, model.NumLODs)
						err = e
						for k, modelLod := range modelLods {
							modelLODOut := ModelLOD{}
							streamInternal.Seek(start+(int32(k)*size), streamInternal.Begin)

							streamInternal.Seek(modelLod.MeshOffset, streamInternal.Position)
							//callback
							func() {
								start := streamInternal.Position
								size := int32(9) //internal.SizeOf(&meshHeader{}) //vtx ignores trailing byte 4-byte alignment

								meshes, e := reader.readMeshes(start, modelLod.NumMeshes)
								err = e
								for l, mesh := range meshes {
									meshOut := Mesh{}
									streamInternal.Seek(start+(int32(l)*size), streamInternal.Begin)

									streamInternal.Seek(mesh.StripGroupHeaderOffset, streamInternal.Position)
									//callback
									func() {
										start := streamInternal.Position
										size := int32(25) //internal.SizeOf(&stripGroupHeader{}) //vtx ignores trailing byte 4-byte alignment

										stripGroups, e := reader.readStripGroups(start, mesh.NumStripGroups)
										err = e
										for m, stripGroup := range stripGroups {
											stripGroupOut := StripGroup{}
											streamInternal.Seek(start+(int32(m)*size), streamInternal.Begin)

											//callback
											func() {
												start := streamInternal.Position
												size := int32(15) //internal.SizeOf(&Strip{})

												stripGroupOut.Vertexes, e = reader.readVertices(start+stripGroup.VertOffset, stripGroup.NumVerts)
												err = e
												stripGroupOut.Indices, e = reader.readIndices(start+stripGroup.IndexOffset, stripGroup.NumIndices)
												err = e
												stripGroupOut.Strips, e = reader.readStrips(start+stripGroup.StripOffset, stripGroup.NumStrips)
												err = e

												for n := range stripGroupOut.Strips {
													streamInternal.Seek(start+(int32(n)*size), streamInternal.Begin)
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

// readHeader reads studiohdr header information
func (reader *Reader) readHeader() (header, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(reader.buf[:headerSize]), binary.LittleEndian, &header)

	return header, err
}

// readBodyParts
func (reader *Reader) readBodyParts(offset int32, num int32) ([]bodyPartHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]bodyPartHeader, 0), errors.New("body part data out of bounds")
	}
	ret := make([]bodyPartHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readModels
func (reader *Reader) readModels(offset int32, num int32) ([]modelHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelHeader, 0), errors.New("model data out of bounds")
	}
	ret := make([]modelHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readModelLODs
func (reader *Reader) readModelLODs(offset int32, num int32) ([]modelLODHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]modelLODHeader, 0), errors.New("model lod data out of bounds")
	}
	ret := make([]modelLODHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readMeshes
func (reader *Reader) readMeshes(offset int32, num int32) ([]meshHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]meshHeader, 0), errors.New("mesh data out of bounds")
	}
	ret := make([]meshHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readStripGroups
func (reader *Reader) readStripGroups(offset int32, num int32) ([]stripGroupHeader, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]stripGroupHeader, 0), errors.New("strip group data out of bounds")
	}
	ret := make([]stripGroupHeader, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readIndices
func (reader *Reader) readIndices(offset int32, num int32) ([]uint16, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]uint16, 0), errors.New("indices data out of bounds")
	}
	ret := make([]uint16, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readVertices
func (reader *Reader) readVertices(offset int32, num int32) ([]Vertex, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Vertex, 0), errors.New("vertex data out of bounds")
	}
	ret := make([]Vertex, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// readStrips
func (reader *Reader) readStrips(offset int32, num int32) ([]Strip, error) {
	if !isPropertyValid(offset, num, len(reader.buf)) {
		return make([]Strip, 0), errors.New("strip data out of bounds")
	}
	ret := make([]Strip, num)
	err := binary.Read(bytes.NewBuffer(reader.buf[offset:]), binary.LittleEndian, &ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func isPropertyValid(offset int32, num int32, bufferSize int) bool {
	if num < 1 || offset < 1 {
		return false
	}
	return true
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
