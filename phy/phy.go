package phy

import "github.com/go-gl/mathgl/mgl32"

type Phy struct {
	Header header
	CompactSurfaces []compactSurfaceHeader
	LegacySurfaces []legacySurfaceHeader
	Text string
}

type header struct {
	Size int32
	Id int32
	SolidCount int32
	CheckSum int32
}

type compactSurfaceHeader struct {
	Size int32
	VPhysicsID int32
	Version int16
	ModelType int16
	SurfaceSize int32
	DragAxisAreas mgl32.Vec3
	AxisMapSize int32
}

type legacySurfaceHeader struct {
	Size int32
	MassCenter mgl32.Vec3
	RotationInertia mgl32.Vec3
	UpperLimitRadius float32
	BVMaxDeviation_ByteSize int32 // bit vector; split 8:24
	OffsetLedgeTreeRoot int32
	_ [3]int32

}