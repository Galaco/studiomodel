package phy

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/galaco/studiomodel/internal"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	// VPhysicsID is the expected CompactSurfaceHeader.VPhysicsID ("VPHY" in little-endian).
	// Older files without it store the compact surface directly after the solid's size.
	VPhysicsID = 0x59485056
	// ModelTypeCompactSurface is the only supported CompactSurfaceHeader.ModelType (IVP compact surface, not MOPP)
	ModelTypeCompactSurface = 0
)

// Reader
type Reader struct {
}

// Read parses a stream to a Phy struct
func (reader *Reader) Read(stream io.Reader) (*Phy, error) {
	buf, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}

	// Read header
	header, err := internal.ReadStruct[Header](buf, 0, "header")
	if err != nil {
		return nil, fmt.Errorf("failed to read PHY header: %w", err)
	}
	if header.Size < int32(binary.Size(header)) || int64(header.Size) > int64(len(buf)) {
		return nil, fmt.Errorf("invalid PHY header size: %d", header.Size)
	}
	if header.SolidCount < 0 {
		return nil, fmt.Errorf("invalid negative solid count: %d", header.SolidCount)
	}

	// Solids follow the header back to back. Each read is bounds checked, so a corrupt count fails rather than
	// allocating.
	offset := int64(header.Size)
	var solids []Solid
	for i := int32(0); i < header.SolidCount; i++ {
		solid, next, err := reader.readSolid(buf, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to read solid %d at offset %d: %w", i, offset, err)
		}
		solids = append(solids, solid)
		offset = next
	}

	return &Phy{
		Header: header,
		Solids: solids,
		Text:   strings.TrimRight(string(buf[offset:]), "\x00"),
	}, nil
}

// readSolid reads the solid at offset, and returns the offset of whatever follows it
func (reader *Reader) readSolid(buf []byte, offset int64) (Solid, int64, error) {
	solid := Solid{}

	var err error
	solid.Header, err = internal.ReadStruct[CompactSurfaceHeader](buf, offset, "compact surface header")
	if err != nil {
		return solid, 0, err
	}
	// Size excludes the size field itself. Confine every other read to this solid.
	end := offset + 4 + int64(solid.Header.Size)
	if err := internal.CheckRange(buf, offset, end-offset, "solid"); err != nil {
		return solid, 0, err
	}
	data := buf[:end]

	surfaceStart := offset + int64(binary.Size(CompactSurfaceHeader{}))
	switch {
	case solid.Header.VPhysicsID != VPhysicsID:
		// Older files have no compact surface header: the compact surface follows the size
		solid.Header = CompactSurfaceHeader{Size: solid.Header.Size}
		surfaceStart = offset + 4
	case solid.Header.ModelType != ModelTypeCompactSurface:
		return solid, 0, fmt.Errorf("unsupported collision model type: %d", solid.Header.ModelType)
	}
	solid.Surface, err = internal.ReadStruct[CompactSurface](data, surfaceStart, "compact surface")
	if err != nil {
		return solid, 0, err
	}

	// Ledges follow the surface back to back, up to the vertices they all share
	ledgeHeaderSize := int64(binary.Size(LedgeHeader{}))
	triangleSize := int64(binary.Size(Triangle{}))
	ledgeStart := surfaceStart + int64(binary.Size(CompactSurface{}))
	first, err := internal.ReadStruct[LedgeHeader](data, ledgeStart, "ledge")
	if err != nil {
		return solid, 0, err
	}
	verticesStart := ledgeStart + int64(first.PointOffset)
	if verticesStart <= ledgeStart {
		return solid, 0, fmt.Errorf("invalid ledge vertex offset: %d", first.PointOffset)
	}

	numVertices := 0
	pos := ledgeStart
	for pos < verticesStart {
		ledgeHeader, err := internal.ReadStruct[LedgeHeader](data, pos, "ledge")
		if err != nil {
			return solid, 0, err
		}
		if pos+int64(ledgeHeader.PointOffset) != verticesStart {
			return solid, 0, fmt.Errorf("ledge %d at offset %d does not share the solid's vertices", len(solid.Ledges), pos)
		}
		triangles, err := internal.ReadSlice[Triangle](data, pos+ledgeHeaderSize, int64(ledgeHeader.NumTriangles), "triangles")
		if err != nil {
			return solid, 0, err
		}
		for _, t := range triangles {
			numVertices = max(numVertices, int(t.V1)+1, int(t.V2)+1, int(t.V3)+1)
		}
		solid.Ledges = append(solid.Ledges, Ledge{
			Header:    ledgeHeader,
			Triangles: triangles,
		})
		pos += ledgeHeaderSize + triangleSize*int64(len(triangles))
	}
	if pos != verticesStart {
		return solid, 0, fmt.Errorf("ledges overrun the solid's vertices: ended at %d, vertices start at %d", pos, verticesStart)
	}

	solid.Vertices, err = internal.ReadSlice[mgl32.Vec4](data, verticesStart, int64(numVertices), "vertices")
	if err != nil {
		return solid, 0, err
	}

	return solid, end, nil
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
