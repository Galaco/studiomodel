package phy

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

// Read parses a stream to a Phy struct
func (reader *Reader) Read(stream io.Reader) (*Phy, error) {
	var buf []byte
	byteBuf := bytes.Buffer{}
	_, err := byteBuf.ReadFrom(stream)
	if err != nil {
		return nil, err
	}
	buf = byteBuf.Bytes()

	// Read header
	header, err := reader.readHeader(buf)
	if err != nil {
		return nil, err
	}

	offset := int32(0)
	if header.SolidCount == 1 {
		return nil, err
	}

	//bodyparts
	offset += int32(unsafe.Sizeof(header))
	compacts, legacys, offsets, err := reader.readSolids(buf, offset, header.SolidCount)
	if err != nil {
		return nil, err
	}


	faceHeaders, faces, vertices, err := reader.readTriangles(buf, offsets)
	if err != nil {
		return nil, err
	}

	return &Phy{
		Header:              header,
		CompactSurfaces:     compacts,
		LegacySurfaces:      legacys,
		TriangleFaceHeaders: faceHeaders,
		TriangleFaces:       faces,
		Vertices:            vertices,
	}, nil
}

// readHeader reads phy header information
func (reader *Reader) readHeader(buf []byte) (header, error) {
	header := header{}
	headerSize := unsafe.Sizeof(header)

	err := binary.Read(bytes.NewBuffer(buf[:headerSize]), binary.LittleEndian, &header)

	return header, err
}

// readSolids reads compact and legacy entries
func (reader *Reader) readSolids(buf []byte, offset int32, num int32) ([]compactSurfaceHeader, []legacySurfaceHeader, []int, error) {
	compacts := make([]compactSurfaceHeader, num)
	legacys := make([]legacySurfaceHeader, num)
	offsets := make([]int, num)
	compactSize := int32(unsafe.Sizeof(compactSurfaceHeader{}))
	legacySize := int32(unsafe.Sizeof(legacySurfaceHeader{}))

	for i := int32(0); i < num; i++ {
		//compact
		err := binary.Read(bytes.NewBuffer(buf[offset:offset+compactSize]), binary.LittleEndian, &compacts[i])
		if err != nil {
			return compacts, legacys, offsets, err
		}
		offsets[i] = int(offset + compactSize + legacySize)

		//legacy
		err = binary.Read(bytes.NewBuffer(buf[offset + compactSize:offset + compactSize+legacySize]), binary.LittleEndian, &legacys[i])
		if err != nil {
			return compacts, legacys, offsets, err
		}

		offset += compacts[i].Size + 4 // ?



	}

	return compacts, legacys, offsets, nil
}

// readTriangles
func (reader *Reader) readTriangles(buf []byte, initialOffsets []int) ([]triangleFaceHeader, []triangleFace, []mgl32.Vec4, error) {
	headers := make([]triangleFaceHeader, 0)
	triangles := make([]triangleFace, 0)
	vertices := make([]mgl32.Vec4, 0)
	headerSize := int(unsafe.Sizeof(triangleFaceHeader{}))
	triangleSize := int(unsafe.Sizeof(triangleFace{}))
	vertexSize := int(unsafe.Sizeof(mgl32.Vec4{}))

	for i := 0; i < len(initialOffsets); i++{
		offset := initialOffsets[i]
		// Read header
		header := triangleFaceHeader{}
		if err := binary.Read(bytes.NewBuffer(buf[offset:offset+headerSize]), binary.LittleEndian, &header); err != nil {
			return nil, nil, nil, err
		}
		vertexDataOffset := offset + int(header.OffsetToVertices)


		// Read triangles referenced in header
		headerTriangles := make([]triangleFace, header.FaceCount)
		if err := binary.Read(
			bytes.NewBuffer(buf[offset+headerSize:offset+headerSize+triangleSize*len(headerTriangles)]),
			binary.LittleEndian,
			&headerTriangles); err != nil {
			return nil, nil, nil, err
		}

		// Prepare the next offset to the next triangle
		offset += (headerSize + (len(headerTriangles) * triangleSize))

		// Discard if set
		if header.DummyFlag & 1 > 0 {
			continue
		}
		headers = append(headers, header)
		triangles = append(triangles, headerTriangles...)

		// calc number of Verts
		numVerts := 0
		for _, t := range headerTriangles {
			if int(t.V1) > numVerts {
				numVerts = int(t.V1)
			}
			if int(t.V2) > numVerts {
				numVerts = int(t.V2)
			}
			if int(t.V3) > numVerts {
				numVerts = int(t.V3)
			}
		}

		// read verts
		triangleVertices := make([]mgl32.Vec4, numVerts + 1)
		if err := binary.Read(
			bytes.NewBuffer(buf[vertexDataOffset:vertexDataOffset + (vertexSize * (numVerts  + 1))]),
			binary.LittleEndian,
			&triangleVertices); err != nil {
			return nil, nil, nil, err
		}

		vertices = append(vertices, triangleVertices...)
	}

	return headers, triangles, vertices, nil
}

// NewReader returns a new reader
func NewReader() *Reader {
	return new(Reader)
}
