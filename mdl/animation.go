package mdl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/galaco/studiomodel/internal"
	"github.com/go-gl/mathgl/mgl32"
)

// AnimDesc and SequenceDesc flags
const (
	// AnimLooping - the last frame matches the first
	AnimLooping = 0x0001
	// AnimSnap - don't interpolate from the previous animation
	AnimSnap = 0x0002
	// AnimDelta - poses are added to another animation's rather than replacing them
	AnimDelta = 0x0004
	// AnimAutoplay
	AnimAutoplay = 0x0008
	// AnimPost
	AnimPost = 0x0010
	// AnimAllZeros - there is no animation data; every bone keeps its default pose
	AnimAllZeros = 0x0020
	// AnimCyclePose - the cycle is taken from a pose parameter
	AnimCyclePose = 0x0080
	// AnimRealtime - the cycle is taken from a real-time clock
	AnimRealtime = 0x0100
	// AnimLocal
	AnimLocal = 0x0200
	// AnimHidden
	AnimHidden = 0x0400
	// AnimOverride
	AnimOverride = 0x0800
	// AnimWorld - blends in world space
	AnimWorld = 0x4000
)

// Per-bone animation track flags (mstudioanim_t)
const (
	trackRawPos  = 0x01 // Vector48
	trackRawRot  = 0x02 // Quaternion48
	trackAnimPos = 0x04 // compressed values
	trackAnimRot = 0x08 // compressed values
	trackDelta   = 0x10
	trackRawRot2 = 0x20 // Quaternion64
)

// AnimBlock locates one block of animation data in the .ani file
// Corresponds to mstudioanimblock_t in studio.h
type AnimBlock struct {
	// DataStart
	DataStart int32
	// DataEnd
	DataEnd int32
}

// animSection is mstudioanimsections_t in studio.h
type animSection struct {
	AnimBlock int32
	AnimIndex int32
}

// Movement is a piece of an animation's root motion
// Corresponds to mstudiomovement_t in studio.h
type Movement struct {
	// EndFrame
	EndFrame int32
	// MotionFlags
	MotionFlags int32
	// V0 - velocity at the start of this piece
	V0 float32
	// V1 - velocity at the end of this piece
	V1 float32
	// Angle - yaw at the end of this piece
	Angle float32
	// Vector - movement relative to this piece's starting angle
	Vector mgl32.Vec3
	// Position - relative to the start of the animation
	Position mgl32.Vec3
}

// IKRule plants an IK chain during part of an animation, such as a foot on the ground
// Corresponds to mstudioikrule_t in studio.h
type IKRule struct {
	Index  int32
	Type   int32
	Chain  int32
	Bone   int32
	Slot   int32
	Height float32
	Radius float32
	Floor  float32
	Pos    mgl32.Vec3
	Q      Quaternion
	// CompressedIKErrorIndex - offset to compressed per-frame error data (relative to this struct), not decoded
	CompressedIKErrorIndex int32
	_                      int32
	IStart                 int32
	IKErrorIndex           int32
	// Start, Peak, Tail and End are cycles: the rule's influence rises from Start to Peak and falls from Tail to End
	Start   float32
	Peak    float32
	Tail    float32
	End     float32
	_       float32
	Contact float32
	Drop    float32
	Top     float32
	_       [3]int32
	// AttachmentIndex - offset to the name of the world attachment (relative to this struct)
	AttachmentIndex int32
	_               [7]int32
}

// BonePose is a bone's position and rotation relative to its parent bone
type BonePose struct {
	Position mgl32.Vec3
	Rotation mgl32.Quat
}

// LoadAnimBlocks reads the .ani file named by AnimBlockName. Animations stored there can only be decoded once it is
// loaded.
func (mdl *Mdl) LoadAnimBlocks(stream io.Reader) error {
	buf, err := io.ReadAll(stream)
	if err != nil {
		return err
	}
	if len(buf) < 4 || binary.LittleEndian.Uint32(buf) != AniMagicNumber {
		return errors.New("not an .ani file: missing IDAG magic number")
	}
	for i, block := range mdl.AnimBlocks[min(1, len(mdl.AnimBlocks)):] {
		if block.DataStart < 0 || block.DataEnd < block.DataStart || int64(block.DataEnd) > int64(len(buf)) {
			return fmt.Errorf("animation block %d (%d to %d) lies outside the .ani file (%d bytes)", i+1, block.DataStart, block.DataEnd, len(buf))
		}
	}
	mdl.ani = buf
	return nil
}

// AniMagicNumber is the expected .ani file ID ("IDAG" in little-endian)
const AniMagicNumber = 0x47414449

// AnimationFrame decodes one frame of an animation into the pose of every bone, relative to its parent. Bones the
// animation doesn't move keep their default pose. For a delta animation (AnimDelta), poses are offsets to add to
// another animation's, and unmoved bones get the identity.
func (mdl *Mdl) AnimationFrame(animation, frame int) ([]BonePose, error) {
	if animation < 0 || animation >= len(mdl.AnimDescs) {
		return nil, fmt.Errorf("animation %d out of range (have %d)", animation, len(mdl.AnimDescs))
	}
	desc := &mdl.AnimDescs[animation]
	if frame < 0 || frame >= int(desc.NumFrames) {
		return nil, fmt.Errorf("frame %d out of range (animation %d has %d)", frame, animation, desc.NumFrames)
	}

	poses := make([]BonePose, len(mdl.Bones))
	for i := range poses {
		if desc.Flags&AnimDelta != 0 {
			poses[i] = BonePose{Rotation: mgl32.QuatIdent()}
		} else {
			poses[i] = BonePose{Position: mdl.Bones[i].Position, Rotation: mdl.Bones[i].Quaternion.Quat()}
		}
	}
	if desc.Flags&AnimAllZeros != 0 {
		return poses, nil
	}

	data, start, localFrame, err := mdl.locateAnimation(animation, frame)
	if err != nil {
		return nil, fmt.Errorf("animation %d frame %d: %w", animation, frame, err)
	}

	// Bone tracks form a list in bone order, each holding the offset of the next. Like the engine, walk it alongside
	// the bones, moving on only when a track matches the bone: a track for a bone that doesn't exist, such as the 255
	// that ends some lists, matches nothing and ends the walk.
	pos, more := start, true
	for bone := 0; bone < len(mdl.Bones) && more; bone++ {
		if err := internal.CheckRange(data, pos, 4, "bone track"); err != nil {
			return nil, fmt.Errorf("animation %d: %w", animation, err)
		}
		if int(data[pos]) != bone {
			continue
		}
		flags, next := data[pos+1], int16(binary.LittleEndian.Uint16(data[pos+2:]))
		if poses[bone], err = decodeTrack(data, pos+4, flags, &mdl.Bones[bone], localFrame); err != nil {
			return nil, fmt.Errorf("animation %d bone %d: %w", animation, bone, err)
		}
		pos += int64(next)
		more = next > 0
	}

	return poses, nil
}

// locateAnimation finds the buffer holding an animation's bone tracks for a frame, where in it they start, and which
// frame of that data to read. Long animations are split into sections, each holding sectionframes frames, with the
// last frame in a section of its own.
func (mdl *Mdl) locateAnimation(animation, frame int) ([]byte, int64, int, error) {
	desc := &mdl.AnimDescs[animation]
	descStart := int64(mdl.Header.LocalAnimationOffset) + int64(animation)*int64(binary.Size(AnimDesc{}))

	block, index := desc.AnimBlock, desc.AnimIndex
	if desc.SectionFrames != 0 {
		section := frame / int(desc.SectionFrames)
		if desc.NumFrames > desc.SectionFrames && frame == int(desc.NumFrames)-1 {
			section = int(desc.NumFrames/desc.SectionFrames) + 1
			frame = 0
		} else {
			frame -= section * int(desc.SectionFrames)
		}
		s, err := internal.ReadStruct[animSection](mdl.buf, descStart+int64(desc.SectionIndex)+int64(section)*int64(binary.Size(animSection{})), "animation section")
		if err != nil {
			return nil, 0, 0, err
		}
		block, index = s.AnimBlock, s.AnimIndex
	}

	switch {
	case block < 0:
		return nil, 0, 0, errors.New("animation data is missing; the model needs recompiling")
	case block == 0:
		// In the mdl, relative to the animation's description
		return mdl.buf, descStart + int64(index), frame, nil
	case int(block) >= len(mdl.AnimBlocks):
		return nil, 0, 0, fmt.Errorf("animation block %d out of range (have %d)", block, len(mdl.AnimBlocks))
	case mdl.ani == nil:
		return nil, 0, 0, fmt.Errorf("animation is in %s, which is not loaded", mdl.AnimBlockName)
	default:
		// In the .ani file, relative to the start of the block, and confined to it
		b := mdl.AnimBlocks[block]
		return mdl.ani[:b.DataEnd], int64(b.DataStart) + int64(index), frame, nil
	}
}

// decodeTrack decodes one bone's position and rotation at a frame from its track data, which starts at pos
func decodeTrack(data []byte, pos int64, flags byte, bone *Bone, frame int) (BonePose, error) {
	delta := flags&trackDelta != 0
	pose := BonePose{Position: bone.Position, Rotation: bone.Quaternion.Quat()}
	if delta {
		pose = BonePose{Rotation: mgl32.QuatIdent()}
	}

	// Rotation: a raw quaternion, or compressed euler angles added to the bone's default
	switch {
	case flags&trackRawRot != 0:
		if err := internal.CheckRange(data, pos, 6, "Quaternion48"); err != nil {
			return pose, err
		}
		pose.Rotation = quaternion48(data[pos:])
	case flags&trackRawRot2 != 0:
		if err := internal.CheckRange(data, pos, 8, "Quaternion64"); err != nil {
			return pose, err
		}
		pose.Rotation = quaternion64(data[pos:])
	case flags&trackAnimRot != 0:
		angles, err := extractValues(data, pos, frame, bone.RotScale)
		if err != nil {
			return pose, fmt.Errorf("rotation: %w", err)
		}
		if !delta {
			angles = angles.Add(bone.Rotation)
		}
		pose.Rotation = angleQuaternion(angles)
	}

	// Position: a raw vector after any raw rotation, or compressed values after any compressed rotation
	switch {
	case flags&trackRawPos != 0:
		at := pos
		if flags&trackRawRot != 0 {
			at += 6
		}
		if flags&trackRawRot2 != 0 {
			at += 8
		}
		if err := internal.CheckRange(data, at, 6, "Vector48"); err != nil {
			return pose, err
		}
		pose.Position = vector48(data[at:])
	case flags&trackAnimPos != 0:
		at := pos
		if flags&trackAnimRot != 0 {
			at += 6
		}
		position, err := extractValues(data, at, frame, bone.PosScale)
		if err != nil {
			return pose, fmt.Errorf("position: %w", err)
		}
		if !delta {
			position = position.Add(bone.Position)
		}
		pose.Position = position
	}

	return pose, nil
}

// extractValues reads three compressed channels at a frame. At pos are three offsets (mstudioanim_valueptr_t), each
// relative to pos and pointing at a channel's values; an offset of 0 means the channel is always 0.
func extractValues(data []byte, pos int64, frame int, scale mgl32.Vec3) (mgl32.Vec3, error) {
	if err := internal.CheckRange(data, pos, 6, "animation value offsets"); err != nil {
		return mgl32.Vec3{}, err
	}
	var out mgl32.Vec3
	for i := range 3 {
		offset := int16(binary.LittleEndian.Uint16(data[pos+2*int64(i):]))
		if offset <= 0 {
			continue
		}
		v, err := extractValue(data, pos+int64(offset), frame)
		if err != nil {
			return out, err
		}
		out[i] = float32(v) * scale[i]
	}
	return out, nil
}

// extractValue reads one compressed channel at a frame. The channel is a run of spans: a header of two bytes, the
// number of values stored (valid) and the number of frames covered (total), followed by those values. Frames past the
// stored values repeat the last one.
func extractValue(data []byte, pos int64, frame int) (int16, error) {
	k := frame
	for {
		if err := internal.CheckRange(data, pos, 2, "animation values"); err != nil {
			return 0, err
		}
		valid, total := int(data[pos]), int(data[pos+1])
		if total == 0 {
			return 0, fmt.Errorf("animation values end before frame %d", frame)
		}
		if total > k {
			index := min(k+1, valid)
			if err := internal.CheckRange(data, pos+2*int64(index), 2, "animation values"); err != nil {
				return 0, err
			}
			return int16(binary.LittleEndian.Uint16(data[pos+2*int64(index):])), nil
		}
		k -= total
		pos += 2 * int64(valid+1)
	}
}

// angleQuaternion converts euler angles in radians (Valve's RadianEuler) to a quaternion
func angleQuaternion(angles mgl32.Vec3) mgl32.Quat {
	sr, cr := math.Sincos(float64(angles[0]) * 0.5)
	sp, cp := math.Sincos(float64(angles[1]) * 0.5)
	sy, cy := math.Sincos(float64(angles[2]) * 0.5)

	srXcp, crXsp := sr*cp, cr*sp
	crXcp, srXsp := cr*cp, sr*sp
	return mgl32.Quat{
		W: float32(crXcp*cy + srXsp*sy),
		V: mgl32.Vec3{
			float32(srXcp*cy - crXsp*sy),
			float32(crXsp*cy + srXcp*sy),
			float32(crXcp*sy - srXsp*cy),
		},
	}
}

// quaternion48 decodes a Quaternion48: x and y in 16 bits, z in 15, and the sign of w, which is implied by the rest
func quaternion48(b []byte) mgl32.Quat {
	x := binary.LittleEndian.Uint16(b)
	y := binary.LittleEndian.Uint16(b[2:])
	zw := binary.LittleEndian.Uint16(b[4:])
	q := mgl32.Vec3{
		float32(int(x)-32768) / 32768,
		float32(int(y)-32768) / 32768,
		float32(int(zw&0x7fff)-16384) / 16384,
	}
	return impliedW(q, zw&0x8000 != 0)
}

// quaternion64 decodes a Quaternion64: x, y and z in 21 bits each, and the sign of w, which is implied by the rest
func quaternion64(b []byte) mgl32.Quat {
	v := binary.LittleEndian.Uint64(b)
	component := func(shift uint) float32 {
		return float32(int((v>>shift)&0x1fffff)-1048576) / 1048576.5
	}
	return impliedW(mgl32.Vec3{component(0), component(21), component(42)}, v>>63 != 0)
}

// impliedW completes a unit quaternion from its vector part and the sign of w
func impliedW(v mgl32.Vec3, negative bool) mgl32.Quat {
	w := float32(math.Sqrt(math.Max(0, 1-float64(v.Dot(v)))))
	if negative {
		w = -w
	}
	return mgl32.Quat{W: w, V: v}
}

// vector48 decodes a Vector48: three half-precision floats
func vector48(b []byte) mgl32.Vec3 {
	return mgl32.Vec3{
		float16(binary.LittleEndian.Uint16(b)),
		float16(binary.LittleEndian.Uint16(b[2:])),
		float16(binary.LittleEndian.Uint16(b[4:])),
	}
}

// float16 converts a half-precision float as Valve's float16 does, which treats the largest exponent as an ordinary
// number rather than infinity
func float16(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exponent := uint32(h>>10) & 0x1f
	mantissa := uint32(h) & 0x3ff
	if exponent == 0 {
		v := float32(mantissa) / 1024 / 16384
		if sign != 0 {
			v = -v
		}
		return v
	}
	return math.Float32frombits(sign | (exponent-15+127)<<23 | mantissa<<13)
}

// readAnimationDetails reads each animation's movements and the IK rules stored in the mdl
func readAnimationDetails(buf []byte, header *Studiohdr, out *Mdl) error {
	descSize := int64(binary.Size(AnimDesc{}))
	out.AnimMovements = make([][]Movement, len(out.AnimDescs))
	out.AnimIKRules = make([][]IKRule, len(out.AnimDescs))
	out.AnimIKRuleAttachments = make([][]string, len(out.AnimDescs))
	for i, desc := range out.AnimDescs {
		descStart := int64(header.LocalAnimationOffset) + int64(i)*descSize
		var err error
		if desc.MovementIndex != 0 {
			if out.AnimMovements[i], err = internal.ReadSlice[Movement](buf, descStart+int64(desc.MovementIndex), int64(desc.NumMovements), "movements"); err != nil {
				return fmt.Errorf("animation %d: %w", i, err)
			}
		}
		// Rules for animations stored in an .ani file are stored there too, and not read
		if desc.IKRuleIndex != 0 {
			rulesStart := descStart + int64(desc.IKRuleIndex)
			if out.AnimIKRules[i], err = internal.ReadSlice[IKRule](buf, rulesStart, int64(desc.NumIKRules), "IK rules"); err != nil {
				return fmt.Errorf("animation %d: %w", i, err)
			}
			if out.AnimIKRuleAttachments[i], err = readNames(buf, rulesStart, out.AnimIKRules[i], func(r IKRule) int32 { return r.AttachmentIndex }, "IK rule attachment"); err != nil {
				return fmt.Errorf("animation %d: %w", i, err)
			}
		}
	}

	if header.AnimblocksNameIndex != 0 {
		var err error
		if out.AnimBlockName, err = internal.CString(buf, int64(header.AnimblocksNameIndex), 256, "animation block file name"); err != nil {
			return err
		}
	}
	var err error
	out.AnimBlocks, err = internal.ReadSlice[AnimBlock](buf, int64(header.AnimblocksIndex), int64(header.AnimblocksCount), "animation blocks")
	return err
}
