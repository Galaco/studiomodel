package phy

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/galaco/studiomodel/internal/testutil"
)

const testText = "solid {\n\"index\" \"0\"\n\"name\" \"root\"\n}\n"

// Offsets of fields in the test file built by buildPhy
const (
	solidCountAt       = 8
	firstSolidAt       = 16
	modelTypeAt        = firstSolidAt + 10
	firstLedgeAt       = firstSolidAt + 32 + 48
	firstNumTrisAt     = firstLedgeAt + 12
	firstPointOffsetAt = firstLedgeAt
)

// buildPhy builds a phy with two solids followed by key-values. The first solid has two convex pieces (ledges) that
// share four vertices, the second has one. The first ledge's ClientData is odd, which older readers skipped.
func buildPhy() []byte {
	b := &testutil.Builder{}
	b.I32(16, 0, 2, 1234) // header: size, id, solidCount, checkSum
	writeSolid(b, [][][3]int16{{{0, 1, 2}}, {{0, 1, 3}, {1, 2, 3}}}, 4)
	writeSolid(b, [][][3]int16{{{0, 1, 2}}}, 3)
	b.Str(testText)
	return b.Bytes
}

func writeSolid(b *testutil.Builder, ledges [][][3]int16, numVertices int) {
	writeSolidWithHeader(b, ledges, numVertices, true)
}

// writeSolidWithHeader writes a solid, with the compact surface header that newer files have or without it
func writeSolidWithHeader(b *testutil.Builder, ledges [][][3]int16, numVertices int, header bool) {
	start := b.Pos()
	b.I32(0) // size, set below
	if header {
		// compact surface header: vphysicsID, version, modelType, surfaceSize, dragAxisAreas, axisMapSize
		b.U8('V', 'P', 'H', 'Y').I16(0x100, ModelTypeCompactSurface).I32(0).F32(1, 1, 1).I32(0)
	}
	// compact surface: massCenter, rotationInertia, upperLimitRadius, max deviation + byte size, ledge tree offset, dummy
	b.F32(0, 0, 0, 1, 1, 1, 1).I32(0, 0).Zero(12)

	verticesAt := b.Pos()
	for _, triangles := range ledges {
		verticesAt += 16 + 16*len(triangles)
	}
	for i, triangles := range ledges {
		pos := b.Pos()
		// pointOffset, clientData, flags (size_div_16 in the high 24 bits), numTriangles, padding
		b.I32(int32(verticesAt-pos), int32(i+1), int32((1+len(triangles))<<8)).I16(int16(len(triangles)), 0)
		for t, triangle := range triangles {
			b.I32(int32(t))
			for _, v := range triangle {
				b.I16(v, 0) // edge: start point index in the low 16 bits
			}
		}
	}
	for v := range numVertices {
		b.F32(float32(v), 0, 0, 0)
	}
	b.Zero(16) // ledge tree
	b.SetI32(start, int32(b.Pos()-start-4))
}

func TestStructSizes(t *testing.T) {
	for name, c := range map[string]struct {
		value any
		want  int
	}{
		"phyheader_t":            {Header{}, 16},
		"compactsurfaceheader_t": {CompactSurfaceHeader{}, 32},
		"IVP_Compact_Surface":    {CompactSurface{}, 48},
		"IVP_Compact_Ledge":      {LedgeHeader{}, 16},
		"IVP_Compact_Triangle":   {Triangle{}, 16},
	} {
		if got := binary.Size(c.value); got != c.want {
			t.Errorf("%s: got %d bytes, want %d", name, got, c.want)
		}
	}
}

func TestRead(t *testing.T) {
	p, err := ReadFromStream(bytes.NewReader(buildPhy()))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Solids) != 2 {
		t.Fatalf("got %d solids, want 2", len(p.Solids))
	}

	first := p.Solids[0]
	if len(first.Ledges) != 2 || len(first.Ledges[0].Triangles) != 1 || len(first.Ledges[1].Triangles) != 2 {
		t.Fatalf("solid 0 ledges: got %+v", first.Ledges)
	}
	if tri := first.Ledges[1].Triangles[1]; tri.V1 != 1 || tri.V2 != 2 || tri.V3 != 3 {
		t.Errorf("solid 0 ledge 1 triangle 1: got %+v", tri)
	}
	if len(first.Vertices) != 4 || first.Vertices[3].X() != 3 {
		t.Errorf("solid 0 vertices: got %v", first.Vertices)
	}

	second := p.Solids[1]
	if len(second.Ledges) != 1 || len(second.Vertices) != 3 {
		t.Errorf("solid 1: got %d ledges, %d vertices", len(second.Ledges), len(second.Vertices))
	}

	if p.Text != testText {
		t.Errorf("text: got %q, want %q", p.Text, testText)
	}
}

// Older files have no compact surface header
func TestReadLegacySolid(t *testing.T) {
	b := &testutil.Builder{}
	b.I32(16, 0, 1, 1234)
	writeSolidWithHeader(b, [][][3]int16{{{0, 1, 2}}, {{0, 2, 3}}}, 4, false)
	b.Str(testText)

	p, err := ReadFromStream(bytes.NewReader(b.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Solids) != 1 || len(p.Solids[0].Ledges) != 2 || len(p.Solids[0].Vertices) != 4 || p.Text != testText {
		t.Errorf("got %+v", p)
	}
}

func TestReadRejectsCorruptData(t *testing.T) {
	for name, corrupt := range map[string]func(b *testutil.Builder){
		"huge solid count":      func(b *testutil.Builder) { b.SetI32(solidCountAt, math.MaxInt32) },
		"negative solid count":  func(b *testutil.Builder) { b.SetI32(solidCountAt, -1) },
		"negative solid size":   func(b *testutil.Builder) { b.SetI32(firstSolidAt, -8) },
		"huge solid size":       func(b *testutil.Builder) { b.SetI32(firstSolidAt, math.MaxInt32) },
		"zero point offset":     func(b *testutil.Builder) { b.SetI32(firstPointOffsetAt, 0) },
		"overlong ledge":        func(b *testutil.Builder) { b.Bytes[firstNumTrisAt] = 100 },
		"unsupported MOPP data": func(b *testutil.Builder) { b.Bytes[modelTypeAt] = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			b := &testutil.Builder{Bytes: buildPhy()}
			corrupt(b)
			if _, err := ReadFromStream(bytes.NewReader(b.Bytes)); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

// Truncated files must fail cleanly rather than panic, or silently read past the end
func TestReadTruncated(t *testing.T) {
	buf := buildPhy()
	textStart := len(buf) - len(testText) - 1
	for n := range textStart {
		if _, err := ReadFromStream(bytes.NewReader(buf[:n])); err == nil {
			t.Errorf("truncated to %d bytes: expected an error", n)
		}
	}
}

func FuzzRead(f *testing.F) {
	f.Add(buildPhy())
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ReadFromStream(bytes.NewReader(data))
	})
}
