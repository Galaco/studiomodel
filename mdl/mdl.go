package mdl

import "github.com/go-gl/mathgl/mgl32"

// Studiohdr is the Mdl header. Contains offsets and info for different data in this file, and associated formats.
// Struct name is kept the same as Valve implementation for readability.
type Studiohdr struct {
	// Id
	Id int32
	// Version
	Version int32
	// Version
	Checksum int32
	// Name
	// 64 char exactly, null byte padded
	Name [64]byte
	// DataLength
	DataLength int32

	// Eyeposition
	Eyeposition mgl32.Vec3
	// Illumposition
	Illumposition mgl32.Vec3
	// HullMin
	HullMin mgl32.Vec3
	// HullMax
	HullMax mgl32.Vec3
	// ViewBBMin
	ViewBBMin mgl32.Vec3
	// ViewBBMax
	ViewBBMax mgl32.Vec3

	// Flags
	Flags int32

	//studio bone
	//BoneCount
	BoneCount int32
	// BoneOffset
	BoneOffset int32
	//studiobonecontroller
	// BoneControllerCount
	BoneControllerCount int32
	// BoneControllerOffset
	BoneControllerOffset int32
	//mstudiohitboxset
	// HitboxCount
	HitboxCount int32
	// HitboxOffset
	HitboxOffset int32
	//mstudioanimdesc
	// LocalAnimationCount
	LocalAnimationCount int32
	// LocalAnimationOffset
	LocalAnimationOffset int32
	//mstudioseqdesc
	// LocalSequenceCount
	LocalSequenceCount int32
	// LocalSequenceOffset
	LocalSequenceOffset int32

	// ActivityListVersion
	ActivityListVersion int32
	// EventsIndexed
	EventsIndexed int32

	//vmt filenames - mstudiotexture
	// TextureCount
	TextureCount int32
	// TextureOffset
	TextureOffset int32

	// TextureDirCount
	TextureDirCount int32
	// TextureDirOffset
	TextureDirOffset int32

	// SkinReferenceCount
	SkinReferenceCount int32
	// SkinReferenceFamilyCount
	SkinReferenceFamilyCount int32
	// SkinReferenceIndex
	SkinReferenceIndex int32

	// mstudiobodyparts
	// BodyPartCount
	BodyPartCount int32
	// BodypartOffset
	BodypartOffset int32

	// mstudioattachment
	// AttachmentCount
	AttachmentCount int32
	// AttachmentOffset
	AttachmentOffset int32

	// LocalNodeCount
	LocalNodeCount int32
	// LocalNodeIndex
	LocalNodeIndex int32
	// LocalNodeNameIndex
	LocalNodeNameIndex int32

	// mstudioflexdesc
	// FlexDescCount
	FlexDescCount int32
	// FlexDescIndex
	FlexDescIndex int32

	// mstudioflexcontroller
	// FlexControllerCount
	FlexControllerCount int32
	// FlexControllerIndex
	FlexControllerIndex int32

	//mstudioflexrule
	// FlexRulesCount
	FlexRulesCount int32
	// FlexRulesIndex
	FlexRulesIndex int32

	//mstudioikchain
	// IkChainCount
	IkChainCount int32
	// IkChainIndex
	IkChainIndex int32

	//mstudiomouth
	// MouthsCount
	MouthsCount int32
	// MouthsIndex
	MouthsIndex int32

	//mstudioposeparamdesc
	// LocalPoseParamCount
	LocalPoseParamCount int32
	// LocalPoseParamIndex
	LocalPoseParamIndex int32

	// SurfacePropertyIndex
	SurfacePropertyIndex int32

	// KeyValueIndex
	KeyValueIndex int32
	// KeyValueCount
	KeyValueCount int32

	// mstudioiklock
	// IkLockCount
	IkLockCount int32
	// IkLockIndex
	IkLockIndex int32

	// Mass
	Mass float32
	// Contents
	Contents int32

	// mstudiomodelgroup
	// IncludeModelCount
	IncludeModelCount int32
	// IncludeModelIndex
	IncludeModelIndex int32

	// VirtualModel
	VirtualModel int32

	// mstudianimblock
	// AnimblocksNameIndex
	AnimblocksNameIndex int32
	// AnimblocksCount
	AnimblocksCount int32
	// AnimblocksIndex
	AnimblocksIndex int32

	// AnimblockModel
	AnimblockModel int32

	// BoneTableNameIndex
	BoneTableNameIndex int32

	// VertexBase
	VertexBase int32
	// OffsetBase
	OffsetBase int32

	// DirectionalDotProduct
	DirectionalDotProduct byte
	// RootLOD
	RootLOD uint8
	// NumAllowedRootLods
	NumAllowedRootLods uint8

	_ byte
	_ int32

	// FlexControllerUICount
	FlexControllerUICount int32
	// FlexControllerUIIndex
	FlexControllerUIIndex int32

	// otional studiohdr2 offset
	// StudioHDR2Index
	StudioHDR2Index int32

	_ int32
}

// Mdl represents the complete parsed data in an Mdl file.
type Mdl struct {
	// Header
	Header Studiohdr
	// Bones
	Bones []Bone
	// BoneControllers
	BoneControllers []BoneController
	// HitboxSet
	HitboxSet []HitboxSet
	// AnimDescs
	AnimDescs []AnimDesc
	// SequenceDescs
	SequenceDescs []SequenceDesc
	// Textures
	Textures []Texture
	// TextureNames
	TextureNames []string //mapped to Textures above.
	// TextureDirs
	TextureDirs []string

	// Some skin stuff here
	// @TODO there may be latter properties
}
