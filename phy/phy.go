package phy

import "github.com/go-gl/mathgl/mgl32"

// Phy
type Phy struct {
	// Header
	Header header
	// CompactSurfaces
	CompactSurfaces []compactSurfaceHeader
	// LegacySurfaces
	LegacySurfaces []legacySurfaceHeader
	// Text
	Text string
	// TriangleFaceHeaders
	TriangleFaceHeaders []triangleFaceHeader
	// TriangleFaces
	TriangleFaces []triangleFace
	// Vertices
	Vertices []mgl32.Vec4
}

// header
type header struct {
	// Size
	Size int32
	// Id
	Id int32
	// SolidCount
	SolidCount int32
	// CheckSum
	CheckSum int32
}

// compactSurfaceHeader
type compactSurfaceHeader struct {
	// Size
	Size int32
	// VPhysicsID
	// Generally the ASCII for "VPHY" in newer files
	VPhysicsID int32
	// Version
	Version int16
	// ModelType
	ModelType int16
	// SurfaceSize
	SurfaceSize int32
	// DragAxisAreas
	DragAxisAreas mgl32.Vec3
	// AxisMapSize
	AxisMapSize int32
}

// legacySurfaceHeader
type legacySurfaceHeader struct {
	// MassCenter
	MassCenter mgl32.Vec3
	// RotationInertia
	RotationInertia mgl32.Vec3
	// UpperLimitRadius
	UpperLimitRadius float32

	// VolumeFull
	VolumeFull int32
	// BVMaxDeviation_ByteSize
	// bit vector; split 8:24
	//BVMaxDeviation_ByteSize int32
	// OffsetLedgeTreeRoot
	//OffsetLedgeTreeRoot int32
	_ [4]int32
}

// triangleFaceHeader
type triangleFaceHeader struct {
	// OffsetToVertices
	OffsetToVertices int32
	// DummyFlag
	DummyFlag int32
	_         int32
	// FaceCount
	FaceCount int32
}

// triangleFace
type triangleFace struct {
	// Id
	Id byte
	_  [3]byte
	// V1
	V1 byte
	_  [3]byte
	// V2
	V2 byte
	_  [3]byte
	// V3
	V3 byte
}
