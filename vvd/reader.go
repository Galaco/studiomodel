package vvd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"io"
	"unsafe"
)

const (
	// VVDMagicNumber is the expected file ID ("IDSV" in little-endian)
	VVDMagicNumber = 0x56534449
	// VVDVersion is the expected file version for Source engine VVD files
	VVDVersion = 4
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

	// Validate minimum file size
	if len(buf) < int(unsafe.Sizeof(header{})) {
		return nil, fmt.Errorf("vvd file too small: %d bytes, expected at least %d", len(buf), unsafe.Sizeof(header{}))
	}

	offset := 0
	// Read header
	header, _, err := reader.readHeader(buf, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD header: %w", err)
	}

	// Validate magic number
	if header.Id != VVDMagicNumber {
		return nil, fmt.Errorf("invalid VVD magic number: got 0x%08X, expected 0x%08X", header.Id, VVDMagicNumber)
	}

	// Validate version
	if header.Version != VVDVersion {
		return nil, fmt.Errorf("unsupported VVD version: got %d, expected %d", header.Version, VVDVersion)
	}

	// Validate LOD count
	if header.NumLODs < 1 || header.NumLODs > MaxNumLods {
		return nil, fmt.Errorf("invalid LOD count: %d (must be between 1 and %d)", header.NumLODs, MaxNumLods)
	}

	// Validate offsets are within file bounds
	if header.FixupTableStart < 0 || header.VertexDataStart < 0 || header.TangentDataStart < 0 {
		return nil, fmt.Errorf("invalid negative offset in VVD header")
	}

	if int(header.FixupTableStart) > len(buf) || int(header.VertexDataStart) > len(buf) || int(header.TangentDataStart) > len(buf) {
		return nil, fmt.Errorf("VVD header offset exceeds file size: file size=%d", len(buf))
	}

	// Read fixups
	fixups, _, err := reader.readFixups(buf, int(header.FixupTableStart), int(header.NumFixups))
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD fixups at offset %d: %w", header.FixupTableStart, err)
	}

	//Read vertices
	vertices, _, err := reader.readVertices(buf, int(header.VertexDataStart), &header)
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD vertices at offset %d: %w", header.VertexDataStart, err)
	}

	//Read tangents
	tangents, _, err := reader.readTangents(buf, int(header.TangentDataStart), &header)
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD tangents at offset %d: %w", header.TangentDataStart, err)
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
	if numFixups < 0 {
		return nil, 0, fmt.Errorf("invalid negative fixup count: %d", numFixups)
	}

	if numFixups == 0 {
		return []fixup{}, offset, nil
	}

	fixupSize := int(unsafe.Sizeof(fixup{}))

	// Validate buffer bounds
	requiredSize := offset + (fixupSize * numFixups)
	if requiredSize > len(buf) {
		return nil, 0, fmt.Errorf("fixup data exceeds buffer: need %d bytes, have %d", requiredSize, len(buf))
	}

	fixups := make([]fixup, numFixups)
	err := binary.Read(bytes.NewBuffer(buf[offset:offset+(fixupSize*numFixups)]), binary.LittleEndian, &fixups)
	if err != nil {
		return nil, 0, err
	}

	return fixups, offset + (fixupSize * numFixups), nil
}

// read vertex data
func (reader *Reader) readVertices(buf []byte, offset int, header *header) ([]vertex, int, error) {
	vertexSize := int(unsafe.Sizeof(vertex{}))

	// Calculate the actual number of vertices stored in the file by using byte offsets
	// The vertices are stored from VertexDataStart to TangentDataStart
	// Note: NumLODVertexes represents vertices needed per LOD (for use with fixups),
	// NOT how many vertices are stored in the file
	vertexDataLength := int(header.TangentDataStart - header.VertexDataStart)
	if vertexDataLength < 0 {
		return nil, 0, fmt.Errorf("invalid vertex data range: TangentDataStart=%d < VertexDataStart=%d",
			header.TangentDataStart, header.VertexDataStart)
	}

	if vertexDataLength%vertexSize != 0 {
		return nil, 0, fmt.Errorf("vertex data length %d is not a multiple of vertex size %d",
			vertexDataLength, vertexSize)
	}

	numVertices := vertexDataLength / vertexSize

	// Validate buffer bounds
	requiredSize := offset + vertexDataLength
	if requiredSize > len(buf) {
		return nil, 0, fmt.Errorf("vertex data exceeds buffer: need %d bytes, have %d", requiredSize, len(buf))
	}

	vertexes := make([]vertex, numVertices)
	err := binary.Read(bytes.NewBuffer(buf[offset:offset+vertexDataLength]), binary.LittleEndian, &vertexes)

	return vertexes, offset + vertexDataLength, err
}

// read tangent data
// NOTE: There is 1 tangent for every vertex
func (reader *Reader) readTangents(buf []byte, offset int, header *header) ([]mgl32.Vec4, int, error) {
	tangentSize := int(unsafe.Sizeof(mgl32.Vec4{}))

	// Calculate number of tangents from vertex count (1 tangent per vertex)
	// Tangents are stored from TangentDataStart to end of vertex data
	vertexDataLength := int(header.TangentDataStart - header.VertexDataStart)
	vertexSize := int(unsafe.Sizeof(vertex{}))
	numTangents := vertexDataLength / vertexSize

	// Calculate tangent data length
	tangentDataLength := numTangents * tangentSize

	// Validate buffer bounds
	requiredSize := offset + tangentDataLength
	if requiredSize > len(buf) {
		return nil, 0, fmt.Errorf("tangent data exceeds buffer: need %d bytes, have %d", requiredSize, len(buf))
	}

	tangents := make([]mgl32.Vec4, numTangents)
	err := binary.Read(bytes.NewBuffer(buf[offset:offset+tangentDataLength]), binary.LittleEndian, &tangents)

	return tangents, offset + tangentDataLength, err
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
