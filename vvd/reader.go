package vvd

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/galaco/studiomodel/internal"
	"github.com/go-gl/mathgl/mgl32"
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
	buf, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	// Validate minimum file size
	headerSize := binary.Size(Header{})
	if len(buf) < headerSize {
		return nil, fmt.Errorf("vvd file too small: %d bytes, expected at least %d", len(buf), headerSize)
	}

	// Read header
	header, err := reader.readHeader(buf)
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
	fixups, err := internal.ReadSlice[Fixup](buf, int64(header.FixupTableStart), int64(header.NumFixups), "fixups")
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD fixups at offset %d: %w", header.FixupTableStart, err)
	}

	// Read vertices. The most detailed LOD uses every vertex in the file.
	vertices, err := internal.ReadSlice[Vertex](buf, int64(header.VertexDataStart), int64(header.NumLODVertexes[0]), "vertices")
	if err != nil {
		return nil, fmt.Errorf("failed to read VVD vertices at offset %d: %w", header.VertexDataStart, err)
	}

	//Read tangents
	// NOTE: There is 1 tangent for every vertex
	tangents, err := internal.ReadSlice[mgl32.Vec4](buf, int64(header.TangentDataStart), int64(len(vertices)), "tangents")
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
func (reader *Reader) readHeader(buf []byte) (Header, error) {
	return internal.ReadStruct[Header](buf, 0, "header")
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
