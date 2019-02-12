package mdl

import "github.com/go-gl/mathgl/mgl32"

type Studiohdr struct {
	Id         int32
	Version    int32
	Checksum   int32
	Name       [64]byte // 64 char exactly, null byte padded
	DataLength int32

	Eyeposition   mgl32.Vec3
	Illumposition mgl32.Vec3
	HullMin       mgl32.Vec3
	HullMax       mgl32.Vec3
	ViewBBMin     mgl32.Vec3
	ViewBBMax     mgl32.Vec3

	Flags int32

	//studio bone
	BoneCount  int32
	BoneOffset int32
	//studiobonecontroller
	BoneControllerCount  int32
	BoneControllerOffset int32
	//mstudiohitboxset
	HitboxCount  int32
	HitboxOffset int32
	//mstudioanimdesc
	LocalAnimationCount  int32
	LocalAnimationOffset int32
	//mstudioseqdesc
	LocalSequenceCount  int32
	LocalSequenceOffset int32

	ActivityListVersion int32
	EventsIndexed       int32

	//vmt filenames - mstudiotexture
	TextureCount  int32
	TextureOffset int32

	TextureDirCount  int32
	TextureDirOffset int32

	SkinReferenceCount       int32
	SkinReferenceFamilyCount int32
	SkinReferenceIndex       int32

	// mstudiobodyparts
	BodyPartCount  int32
	BodypartOffset int32

	// mstudioattachment
	AttachmentCount  int32
	AttachmentOffset int32

	LocalNodeCount     int32
	LocalNodeIndex     int32
	LocalNodeNameIndex int32

	// mstudioflexdesc
	FlexDescCount int32
	FlexDescIndex int32

	// mstudioflexcontroller
	FlexControllerCount int32
	FlexControllerIndex int32

	//mstudioflexrule
	FlexRulesCount int32
	FlexRulesIndex int32

	//mstudioikchain
	IkChainCount int32
	IkChainIndex int32

	//mstudiomouth
	MouthsCount int32
	MouthsIndex int32

	//mstudioposeparamdesc
	LocalPoseParamCount int32
	LocalPoseParamIndex int32

	SurfacePropertyIndex int32

	KeyValueIndex int32
	KeyValueCount int32

	// mstudioiklock
	IkLockCount int32
	IkLockIndex int32

	Mass     float32
	Contents int32

	// mstudiomodelgroup
	IncludeModelCount int32
	IncludeModelIndex int32

	VirtualModel int32

	// mstudianimblock
	AnimblocksNameIndex int32
	AnimblocksCount     int32
	AnimblocksIndex     int32

	AnimblockModel int32

	BoneTableNameIndex int32

	VertexBase int32
	OffsetBase int32

	DirectionalDotProduct byte
	RootLOD               uint8
	NumAllowedRootLods    uint8

	_ byte
	_ int32

	FlexControllerUICount int32
	FlexControllerUIIndex int32

	// otional studiohdr2 offset
	StudioHDR2Index int32

	_ int32
}

type Mdl struct {
	Header          Studiohdr
	Bones           []Bone
	BoneControllers []BoneController
	HitboxSet       []HitboxSet
	AnimDescs       []AnimDesc
	SequenceDescs   []SequenceDesc
	Textures        []Texture
	TextureNames    []string //mapped to Textures above.
	TextureDirs     []string
	// Some skin stuff here
}
