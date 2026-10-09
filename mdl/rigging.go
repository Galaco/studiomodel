package mdl

import (
	"encoding/binary"
	"fmt"

	"github.com/galaco/studiomodel/internal"
	"github.com/go-gl/mathgl/mgl32"
)

// Hitbox is a box attached to a bone, used for hit detection
// Corresponds to mstudiobbox_t in studio.h
type Hitbox struct {
	// Bone
	Bone int32
	// Group - hit group, such as head or chest
	Group int32
	// BBMin
	BBMin mgl32.Vec3
	// BBMax
	BBMax mgl32.Vec3
	// NameIndex - offset to the hitbox name (relative to this struct); most hitboxes are unnamed
	NameIndex int32
	// Angles - orientation of the box relative to its bone (CS:GO only)
	Angles mgl32.Vec3
	// CapsuleRadius - radius when the hitbox is a capsule (CS:GO only)
	CapsuleRadius float32
	_             [4]int32
}

// Attachment is a named point relative to a bone, such as a muzzle or a hand
// Corresponds to mstudioattachment_t in studio.h
type Attachment struct {
	// NameIndex - offset to the attachment name (relative to this struct)
	NameIndex int32
	// Flags
	Flags uint32
	// LocalBone
	LocalBone int32
	// Local - transform from the bone
	Local Matrix3x4
	_     [8]int32
}

// PoseParameter is a named animation input, such as aim yaw or move speed
// Corresponds to mstudioposeparamdesc_t in studio.h
type PoseParameter struct {
	// NameIndex - offset to the pose parameter name (relative to this struct)
	NameIndex int32
	// Flags
	Flags int32
	// Start - value at the start of the range
	Start float32
	// End - value at the end of the range
	End float32
	// Loop - looping range, 0 for no looping, 360 for rotations
	Loop float32
}

// IKChain is a chain of bones solved by inverse kinematics, such as a leg
// Corresponds to mstudioikchain_t in studio.h
type IKChain struct {
	// NameIndex - offset to the chain name (relative to this struct)
	NameIndex int32
	// LinkType
	LinkType int32
	// NumLinks
	NumLinks int32
	// LinkIndex - byte offset from start of this struct to first link
	LinkIndex int32
}

// IKLink is one bone of an IKChain
// Corresponds to mstudioiklink_t in studio.h
type IKLink struct {
	// Bone
	Bone int32
	// KneeDir - ideal bending direction
	KneeDir mgl32.Vec3
	_       [3]float32
}

// IKChainData contains a parsed IK chain with its name and links
type IKChainData struct {
	Header IKChain
	// Name
	Name string
	// Links
	Links []IKLink
}

// IKLock holds an IK chain in place while sequences play
// Corresponds to mstudioiklock_t in studio.h
type IKLock struct {
	// Chain
	Chain int32
	// PosWeight
	PosWeight float32
	// LocalQWeight
	LocalQWeight float32
	// Flags
	Flags int32
	_     [4]int32
}

// modelGroup is mstudiomodelgroup_t in studio.h
type modelGroup struct {
	// LabelIndex - offset to the label (relative to this struct)
	LabelIndex int32
	// NameIndex - offset to the model name (relative to this struct)
	NameIndex int32
}

// IncludeModel is another model whose sequences and animations this model uses
type IncludeModel struct {
	// Label
	Label string
	// Name - path of the included model
	Name string
}

// Studiohdr2 is the secondary header. Its offsets are relative to itself, except SrcBoneTransformIndex, which is
// relative to the main header.
// Corresponds to studiohdr2_t in studio.h
type Studiohdr2 struct {
	// NumSrcBoneTransforms
	NumSrcBoneTransforms int32
	// SrcBoneTransformIndex
	SrcBoneTransformIndex int32
	// IllumPositionAttachmentIndex - attachment used as the illumination centre
	IllumPositionAttachmentIndex int32
	// MaxEyeDeflection
	MaxEyeDeflection float32
	// LinearBoneIndex - offset to a copy of the bones laid out by field, for the engine's benefit
	LinearBoneIndex int32
	// NameIndex - offset to the model's full name, which the main header truncates at 64 characters
	NameIndex int32
	// BoneFlexDriverCount
	BoneFlexDriverCount int32
	// BoneFlexDriverIndex
	BoneFlexDriverIndex int32
	_                   [56]int32
}

// SrcBoneTransform is a transform studiomdl applied to a bone of a source model
// Corresponds to mstudiosrcbonetransform_t in studio.h
type SrcBoneTransform struct {
	// NameIndex - offset to the bone name (relative to this struct)
	NameIndex int32
	// PreTransform
	PreTransform Matrix3x4
	// PostTransform
	PostTransform Matrix3x4
}

// Bone.ProcType values
const (
	// ProcAxisInterp
	ProcAxisInterp = 1
	// ProcQuatInterp
	ProcQuatInterp = 2
	// ProcAimAtBone
	ProcAimAtBone = 3
	// ProcAimAtAttachment
	ProcAimAtAttachment = 4
	// ProcJiggle
	ProcJiggle = 5
)

// ProceduralBone is the rule that moves one procedural bone. The rule matching Type is set; types this package does
// not read (such as CS:GO twist bones) set none.
type ProceduralBone struct {
	// Bone - index into Bones
	Bone int
	// Type - one of the Proc constants
	Type int32
	// AxisInterp
	AxisInterp *AxisInterpBone
	// QuatInterp
	QuatInterp *QuatInterpBone
	// AimAt - for ProcAimAtBone and ProcAimAtAttachment
	AimAt *AimAtBone
	// Jiggle
	Jiggle *JiggleBone
	// Boing - for jiggle bones with JiggleIsBoing
	Boing *JiggleBoing
}

// AxisInterpBone
// Corresponds to mstudioaxisinterpbone_t in studio.h
type AxisInterpBone struct {
	// Control - bone whose rotation drives this one
	Control int32
	// Axis
	Axis int32
	// Pos
	Pos [6]mgl32.Vec3
	// Quat
	Quat [6]Quaternion
}

// quatInterpBoneHeader is mstudioquatinterpbone_t in studio.h
type quatInterpBoneHeader struct {
	// Control
	Control int32
	// NumTriggers
	NumTriggers int32
	// TriggerIndex - byte offset from start of this struct to first trigger
	TriggerIndex int32
}

// QuatInterpBone moves a bone by blending between poses, according to how close a control bone is to each trigger
type QuatInterpBone struct {
	// Control - bone whose rotation drives this one
	Control int32
	// Triggers
	Triggers []QuatInterpTrigger
}

// QuatInterpTrigger
// Corresponds to mstudioquatinterpinfo_t in studio.h
type QuatInterpTrigger struct {
	// InvTolerance
	InvTolerance float32
	// Trigger - control bone rotation that selects this pose
	Trigger Quaternion
	// Pos
	Pos mgl32.Vec3
	// Quat
	Quat Quaternion
}

// AimAtBone points a bone at another bone or attachment
// Corresponds to mstudioaimatbone_t in studio.h
type AimAtBone struct {
	// Parent
	Parent int32
	// Aim - bone or attachment to aim at
	Aim int32
	// AimVector
	AimVector mgl32.Vec3
	// UpVector
	UpVector mgl32.Vec3
	// BasePos
	BasePos mgl32.Vec3
}

// JiggleBone.Flags values
const (
	JiggleIsFlexible          = 0x01
	JiggleIsRigid             = 0x02
	JiggleHasYawConstraint    = 0x04
	JiggleHasPitchConstraint  = 0x08
	JiggleHasAngleConstraint  = 0x10
	JiggleHasLengthConstraint = 0x20
	JiggleHasBaseSpring       = 0x40
	JiggleIsBoing             = 0x80
)

// JiggleBone is a bone that springs around its rest position. Angles are in radians.
// Corresponds to mstudiojigglebone_t in studio.h
type JiggleBone struct {
	// Flags - Jiggle constants
	Flags int32

	// Length - distance along the bone from its base to its tip
	Length float32
	// TipMass
	TipMass float32

	YawStiffness   float32
	YawDamping     float32
	PitchStiffness float32
	PitchDamping   float32
	AlongStiffness float32
	AlongDamping   float32

	// AngleLimit - maximum deflection of the tip
	AngleLimit float32

	MinYaw      float32
	MaxYaw      float32
	YawFriction float32
	YawBounce   float32

	MinPitch      float32
	MaxPitch      float32
	PitchFriction float32
	PitchBounce   float32

	BaseMass            float32
	BaseStiffness       float32
	BaseDamping         float32
	BaseMinLeft         float32
	BaseMaxLeft         float32
	BaseLeftFriction    float32
	BaseMinUp           float32
	BaseMaxUp           float32
	BaseUpFriction      float32
	BaseMinForward      float32
	BaseMaxForward      float32
	BaseForwardFriction float32
}

// JiggleBoing follows a JiggleBone that has JiggleIsBoing. Older models end the jiggle bone without it.
type JiggleBoing struct {
	ImpactSpeed float32
	ImpactAngle float32
	DampingRate float32
	Frequency   float32
	Amplitude   float32
}

// readHitboxes reads the hitboxes of each hitbox set, and their names
func readHitboxes(buf []byte, header *Studiohdr, sets []HitboxSet) ([][]Hitbox, [][]string, error) {
	setSize := int64(binary.Size(HitboxSet{}))
	hitboxSize := int64(binary.Size(Hitbox{}))
	hitboxes := make([][]Hitbox, len(sets))
	names := make([][]string, len(sets))
	for i, set := range sets {
		// HitboxIndex is relative to the start of this set
		start := int64(header.HitboxOffset) + int64(i)*setSize + int64(set.HitboxIndex)
		var err error
		if hitboxes[i], err = internal.ReadSlice[Hitbox](buf, start, int64(set.NumHitboxes), "hitboxes"); err != nil {
			return nil, nil, fmt.Errorf("hitbox set %d: %w", i, err)
		}

		// Name offsets are relative to the hitbox, except in some v44 files where they are relative to the file.
		// The engine never reads them, so a name that resolves neither way is left empty.
		names[i] = make([]string, len(hitboxes[i]))
		for j, hitbox := range hitboxes[i] {
			if hitbox.NameIndex == 0 {
				continue
			}
			for _, offset := range []int64{start + int64(j)*hitboxSize + int64(hitbox.NameIndex), int64(hitbox.NameIndex)} {
				if name, err := internal.CString(buf, offset, 256, "hitbox name"); err == nil {
					names[i][j] = name
					break
				}
			}
		}
	}
	return hitboxes, names, nil
}

// readProceduralBones reads the rule of every procedural bone. Rule offsets are relative to their bone.
func readProceduralBones(buf []byte, header *Studiohdr, bones []Bone) ([]ProceduralBone, error) {
	boneSize := int64(binary.Size(Bone{}))
	var out []ProceduralBone
	for i, bone := range bones {
		if bone.ProcType == 0 || bone.ProcIndex == 0 {
			continue
		}
		start := int64(header.BoneOffset) + int64(i)*boneSize + int64(bone.ProcIndex)
		proc := ProceduralBone{Bone: i, Type: bone.ProcType}

		var err error
		switch bone.ProcType {
		case ProcAxisInterp:
			proc.AxisInterp, err = readPointer[AxisInterpBone](buf, start, "axis interp bone")
		case ProcQuatInterp:
			var h quatInterpBoneHeader
			if h, err = internal.ReadStruct[quatInterpBoneHeader](buf, start, "quat interp bone"); err == nil {
				proc.QuatInterp = &QuatInterpBone{Control: h.Control}
				proc.QuatInterp.Triggers, err = internal.ReadSlice[QuatInterpTrigger](buf, start+int64(h.TriggerIndex), int64(h.NumTriggers), "quat interp triggers")
			}
		case ProcAimAtBone, ProcAimAtAttachment:
			proc.AimAt, err = readPointer[AimAtBone](buf, start, "aim at bone")
		case ProcJiggle:
			if proc.Jiggle, err = readPointer[JiggleBone](buf, start, "jiggle bone"); err == nil && proc.Jiggle.Flags&JiggleIsBoing != 0 {
				proc.Boing, err = readPointer[JiggleBoing](buf, start+int64(binary.Size(JiggleBone{})), "jiggle bone boing")
			}
		}
		if err != nil {
			return nil, fmt.Errorf("bone %d: %w", i, err)
		}
		out = append(out, proc)
	}
	return out, nil
}

// readPointer reads a T at offset and returns a pointer to it
func readPointer[T any](buf []byte, offset int64, name string) (*T, error) {
	v, err := internal.ReadStruct[T](buf, offset, name)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// readIKChains reads the IK chains with their names and links
func readIKChains(buf []byte, header *Studiohdr) ([]IKChainData, error) {
	chains, err := internal.ReadSlice[IKChain](buf, int64(header.IkChainIndex), int64(header.IkChainCount), "IK chains")
	if err != nil {
		return nil, err
	}
	names, err := readNames(buf, int64(header.IkChainIndex), chains, func(c IKChain) int32 { return c.NameIndex }, "IK chain")
	if err != nil {
		return nil, err
	}

	chainSize := int64(binary.Size(IKChain{}))
	out := make([]IKChainData, len(chains))
	for i, chain := range chains {
		// LinkIndex is relative to the start of this chain
		start := int64(header.IkChainIndex) + int64(i)*chainSize + int64(chain.LinkIndex)
		links, err := internal.ReadSlice[IKLink](buf, start, int64(chain.NumLinks), "IK links")
		if err != nil {
			return nil, fmt.Errorf("IK chain %d: %w", i, err)
		}
		out[i] = IKChainData{Header: chain, Name: names[i], Links: links}
	}
	return out, nil
}

// readIncludeModels reads the labels and names of included models
func readIncludeModels(buf []byte, header *Studiohdr) ([]IncludeModel, error) {
	groups, err := internal.ReadSlice[modelGroup](buf, int64(header.IncludeModelIndex), int64(header.IncludeModelCount), "include models")
	if err != nil {
		return nil, err
	}
	labels, err := readNames(buf, int64(header.IncludeModelIndex), groups, func(g modelGroup) int32 { return g.LabelIndex }, "include model label")
	if err != nil {
		return nil, err
	}
	names, err := readNames(buf, int64(header.IncludeModelIndex), groups, func(g modelGroup) int32 { return g.NameIndex }, "include model")
	if err != nil {
		return nil, err
	}

	out := make([]IncludeModel, len(groups))
	for i := range groups {
		out[i] = IncludeModel{Label: labels[i], Name: names[i]}
	}
	return out, nil
}

// readHeader2 reads the secondary header, if there is one, with its source bone transforms and the model's long name
func readHeader2(buf []byte, header *Studiohdr, out *Mdl) error {
	if header.StudioHDR2Index == 0 {
		return nil
	}
	start := int64(header.StudioHDR2Index)
	header2, err := readPointer[Studiohdr2](buf, start, "studiohdr2")
	if err != nil {
		return err
	}
	out.Header2 = header2

	// Unlike the secondary header's other offsets, this one is relative to the main header
	transformsStart := int64(header2.SrcBoneTransformIndex)
	if out.SrcBoneTransforms, err = internal.ReadSlice[SrcBoneTransform](buf, transformsStart, int64(header2.NumSrcBoneTransforms), "source bone transforms"); err != nil {
		return err
	}
	if out.SrcBoneTransformNames, err = readNames(buf, transformsStart, out.SrcBoneTransforms, func(t SrcBoneTransform) int32 { return t.NameIndex }, "source bone transform"); err != nil {
		return err
	}

	if header2.NameIndex != 0 {
		if out.Name, err = internal.CString(buf, start+int64(header2.NameIndex), 1024, "studiohdr2 name"); err != nil {
			return err
		}
	}
	return nil
}

// readRigging reads the sections that describe how the model moves and is hit: hitboxes, attachments, pose
// parameters, procedural bones, IK, included models and the secondary header
func readRigging(buf []byte, header *Studiohdr, out *Mdl) error {
	var err error
	if out.Hitboxes, out.HitboxNames, err = readHitboxes(buf, header, out.HitboxSet); err != nil {
		return err
	}

	if out.Attachments, err = internal.ReadSlice[Attachment](buf, int64(header.AttachmentOffset), int64(header.AttachmentCount), "attachments"); err != nil {
		return err
	}
	if out.AttachmentNames, err = readNames(buf, int64(header.AttachmentOffset), out.Attachments, func(a Attachment) int32 { return a.NameIndex }, "attachment"); err != nil {
		return err
	}

	if out.PoseParameters, err = internal.ReadSlice[PoseParameter](buf, int64(header.LocalPoseParamIndex), int64(header.LocalPoseParamCount), "pose parameters"); err != nil {
		return err
	}
	if out.PoseParameterNames, err = readNames(buf, int64(header.LocalPoseParamIndex), out.PoseParameters, func(p PoseParameter) int32 { return p.NameIndex }, "pose parameter"); err != nil {
		return err
	}

	if out.ProceduralBones, err = readProceduralBones(buf, header, out.Bones); err != nil {
		return err
	}
	if out.IKChains, err = readIKChains(buf, header); err != nil {
		return err
	}
	if out.IKLocks, err = internal.ReadSlice[IKLock](buf, int64(header.IkLockIndex), int64(header.IkLockCount), "IK locks"); err != nil {
		return err
	}
	if out.IncludeModels, err = readIncludeModels(buf, header); err != nil {
		return err
	}

	return readHeader2(buf, header, out)
}
