package phy

import "github.com/go-gl/mathgl/mgl32"

// Phy
type Phy struct {
	// Header
	Header Header
	// Solids is one entry per collision solid. Ragdolls have one solid per physics bone.
	Solids []Solid
	// Text is the key-values block that follows the solids
	// (solid, ragdollconstraint, collisionrules and editparams sections)
	Text string
}

// Header
type Header struct {
	// Size
	Size int32
	// Id
	Id int32
	// SolidCount
	SolidCount int32
	// CheckSum
	CheckSum int32
}

// Solid is one collision solid: one or more convex pieces (ledges) that share a vertex array
type Solid struct {
	// Header
	Header CompactSurfaceHeader
	// Surface
	Surface CompactSurface
	// Ledges are the convex pieces of this solid
	Ledges []Ledge
	// Vertices are shared by all ledges of this solid, and indexed by their triangles.
	// They are in IVP space (metres, with axes that differ from Source), not Source units.
	Vertices []mgl32.Vec4
}

// CompactSurfaceHeader
type CompactSurfaceHeader struct {
	// Size of the solid's data after this field
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

// CompactSurface is IVP_Compact_Surface
type CompactSurface struct {
	// MassCenter
	MassCenter mgl32.Vec3
	// RotationInertia
	RotationInertia mgl32.Vec3
	// UpperLimitRadius
	UpperLimitRadius float32
	// MaxDeviationAndByteSize
	// bit field; max_factor_surface_deviation in the low 8 bits, byte_size in the high 24
	MaxDeviationAndByteSize int32
	// OffsetLedgeTreeRoot
	// relative to the start of this struct
	OffsetLedgeTreeRoot int32
	_                   [3]int32
}

// Ledge is one convex piece of a solid
type Ledge struct {
	// Header
	Header LedgeHeader
	// Triangles
	Triangles []Triangle
}

// LedgeHeader is IVP_Compact_Ledge
type LedgeHeader struct {
	// PointOffset
	// offset from the start of this ledge to the solid's vertices
	PointOffset int32
	// ClientData
	ClientData int32
	// Flags
	// bit field; has_children_flag:2, is_compact_flag:2, dummy:4, size_div_16:24
	Flags int32
	// NumTriangles
	NumTriangles int16
	_            int16
}

// Triangle is IVP_Compact_Triangle
type Triangle struct {
	// Id
	Id byte
	_  [3]byte
	// V1
	V1 uint16
	_  [2]byte
	// V2
	V2 uint16
	_  [2]byte
	// V3
	V3 uint16
	_  [2]byte
}
