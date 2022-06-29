package vvd

import (
	"bytes"
	"encoding/binary"
	"github.com/go-gl/mathgl/mgl32"
	"io"
	"unsafe"
)

// Reader
type Reader struct {
}

// Read parses a stream to a Vvd struct
func (reader *Reader) Read(stream io.Reader) (*Vvd, error) {
	var buf []byte
	byteBuf := bytes.Buffer{}
	_, err := byteBuf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	buf = byteBuf.Bytes()

	offset := 0
	// Read header
	header, _, err := reader.readHeader(buf, offset)
	if err != nil {
		return nil, err
	}

	// Read fixups
	fixups, _, err := reader.readFixups(buf, int(header.FixupTableStart), int(header.NumFixups))
	if err != nil {
		return nil, err
	}

	//Read vertices
	vertices, _, err := reader.readVertices(buf, int(header.VertexDataStart), &header)
	if err != nil {
		return nil, err
	}

	//Read tangents
	tangents, _, err := reader.readTangents(buf, int(header.TangentDataStart), &header)
	if err != nil {
		return nil, err
	}

	return &Vvd{
		Header:   header,
		Fixups:   fixups,
		Vertices: vertices,
		Tangents: tangents,
	}, nil
}

// Reads studiohdr header information
func (reader *Reader) readHeader(buf []byte, offset int) (header, int, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(buf[offset:headerSize]), binary.LittleEndian, &header)

	return header, int(headerSize), err
}

func (reader *Reader) readFixups(buf []byte, offset int, numFixups int) ([]fixup, int, error) {
	fixupSize := int(unsafe.Sizeof(fixup{}))
	fixups := make([]fixup, numFixups)
	if numFixups > 0 {
		err := binary.Read(bytes.NewBuffer(buf[offset:offset+(fixupSize*numFixups)]), binary.LittleEndian, &fixups)
		if err != nil {
			return fixups, 0, err
		}
	}

	return fixups, offset + (fixupSize * numFixups), nil
}

// read vertex data
func (reader *Reader) readVertices(buf []byte, offset int, header *header) ([]vertex, int, error) {
	vertexSize := int(unsafe.Sizeof(vertex{}))
	// Compute number of vertices to read
	numVertices := 0
	for i := int32(0); i < header.NumLODs; i++ {
		numVertices += int(header.NumLODVertexes[i])
	}
	numVertices = int(header.NumLODVertexes[0])
	vertexes := make([]vertex, numVertices)
	err := binary.Read(bytes.NewBuffer(buf[offset:offset+(vertexSize*numVertices)]), binary.LittleEndian, &vertexes)

	return vertexes, offset + (vertexSize * numVertices), err
}

// read tangent data
// NOTE: There is 1 tangent for every vertex
func (reader *Reader) readTangents(buf []byte, offset int, header *header) ([]mgl32.Vec4, int, error) {
	tangentSize := int(unsafe.Sizeof(mgl32.Vec4{}))
	// Compute number of tangents to read
	numTangents := 0
	for i := int32(0); i < header.NumLODs; i++ {
		numTangents += int(header.NumLODVertexes[i])
	}
	numTangents = int(header.NumLODVertexes[0])
	tangents := make([]mgl32.Vec4, numTangents)
	err := binary.Read(bytes.NewBuffer(buf[offset:offset+(tangentSize*numTangents)]), binary.LittleEndian, &tangents)

	return tangents, offset + (tangentSize * numTangents), err
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
