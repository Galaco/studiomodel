package mdl

import (
	"bytes"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/galaco/studiomodel/internal/testutil"
	"github.com/go-gl/mathgl/mgl32"
)

// Animations in the file built by buildAnimatedMdl
const (
	animWalk      = iota // bone 0 raw (Quaternion64, Vector48), bone 1 compressed; with a movement and an IK rule
	animDelta            // delta: bone 1 raw (Quaternion48)
	animIdle             // a track list holding only the 255 terminator
	animSectioned        // 4 frames in sections of 2; the last frame has a section of its own
	animInAni            // in block 1 of the .ani file
	numAnims
)

// Half-precision floats for Vector48s
const (
	half1_5   = 0x3e00
	halfNeg2  = -0x4000 // 0xc000
	half0_25  = 0x3400
	half3     = 0x4200
	half4     = 0x4400
	half8     = 0x4800
	animDesc  = 100 // size of mstudioanimdesc_t
	seqDesc   = 212 // size of mstudioseqdesc_t
	eventSize = 80  // size of mstudioevent_t
)

// buildAnimatedMdl builds buildMdl's model with the animations above, a sequence that blends two of them, and a
// two-node transition graph; and the .ani file animInAni is in
func buildAnimatedMdl() (mdlFile []byte, aniFile []byte) {
	b := buildMdl()
	const child = bonesAt + boneSize
	set := func(at int, values ...float32) {
		for i, v := range values {
			b.SetF32(at+4*i, v)
		}
	}
	str := func(s string) int {
		at := b.Pos()
		b.Str(s)
		return at
	}
	// channel appends one compressed channel's spans, each {valid, total, values...}, and points an offset at it
	channel := func(valuePtr, axis int, spans ...[]int16) {
		b.SetI16(valuePtr+2*axis, int16(b.Pos()-valuePtr))
		for _, span := range spans {
			b.U8(byte(span[0]), byte(span[1])).I16(span[2:]...)
		}
	}

	// bone 1: position (1, 2, 3), position scale 0.5, rotation scale 0.01
	set(child+32, 1, 2, 3)
	set(child+72, 0.5, 0.5, 0.5, 0.01, 0.01, 0.01)

	descs := b.Pos()
	b.Zero(numAnims * animDesc)
	b.SetI32(180, numAnims)
	b.SetI32(184, int32(descs))
	desc := func(anim int) int { return descs + anim*animDesc }
	for anim, name := range []string{"@walk", "@delta", "@idle", "@sectioned", "@ani"} {
		b.SetI32(desc(anim)+4, int32(str(name)-desc(anim)))
		set(desc(anim)+8, 30) // fps
	}
	// fromDesc is an offset from an animation's description to what is appended next
	fromDesc := func(anim int) int32 { return int32(b.Pos() - desc(anim)) }

	// walk: 3 frames
	b.SetI32(desc(animWalk)+16, 3)
	b.SetI32(desc(animWalk)+56, fromDesc(animWalk))
	bone0 := b.Pos()
	b.U8(0, trackRawRot2|trackRawPos).I16(0)
	q64 := uint64(1048576) | uint64(1048576)<<21 | uint64(1677722)<<42 // Quaternion64 (0, 0, 0.6)
	for i := range 8 {
		b.U8(byte(q64 >> (8 * i)))
	}
	b.I16(half1_5, halfNeg2, half0_25)
	b.SetI16(bone0+2, int16(b.Pos()-bone0))
	b.U8(1, trackAnimRot|trackAnimPos).I16(0)
	rot, pos := b.Pos(), b.Pos()+6
	b.Zero(12)
	channel(rot, 0, []int16{3, 3, 10, 20, 30})          // x: one value per frame
	channel(rot, 2, []int16{1, 3, 5})                   // z: 5 throughout
	channel(pos, 0, []int16{2, 3, 4, 6})                // x: 4, 6, then the last value repeats
	channel(pos, 1, []int16{1, 1, 7}, []int16{1, 2, 9}) // y: 7, then 9 for two frames
	b.SetI32(desc(animWalk)+20, 1)
	b.SetI32(desc(animWalk)+24, fromDesc(animWalk))
	b.I32(2, 1).F32(0, 10, 0, 20, 0, 0, 20, 0, 0) // movement
	rule := b.Pos()
	b.SetI32(desc(animWalk)+60, 1)
	b.SetI32(desc(animWalk)+64, fromDesc(animWalk))
	b.Zero(152)
	b.SetI32(rule+12, 1) // bone
	set(rule+76, 0.1, 0.2, 0.8, 0.9)
	b.SetI32(rule+120, int32(str("foot")-rule))

	// delta: bone 1 rotates 90 degrees about z; every other bone gets the identity
	b.SetI32(desc(animDelta)+12, AnimDelta)
	b.SetI32(desc(animDelta)+16, 1)
	b.SetI32(desc(animDelta)+56, fromDesc(animDelta))
	b.U8(1, trackRawRot|trackDelta).I16(0).I16(-32768, -32768, 27969) // Quaternion48 (0, 0, 0.7071)

	// idle: no bone tracks, so every bone keeps its default pose
	b.SetI32(desc(animIdle)+16, 1)
	b.SetI32(desc(animIdle)+56, fromDesc(animIdle))
	b.U8(255, 0).I16(0)

	// sectioned: bone 1's x position is 1.5 and 2 in section 0, 3 in section 1, and 4 in section 3 for the last frame
	b.SetI32(desc(animSectioned)+16, 4)
	b.SetI32(desc(animSectioned)+84, 2)
	b.SetI32(desc(animSectioned)+80, fromDesc(animSectioned))
	sections := b.Pos()
	b.Zero(4 * 8)
	b.SetI32(sections+4, fromDesc(animSectioned))
	b.U8(1, trackAnimPos).I16(0)
	ptr := b.Pos()
	b.Zero(6)
	channel(ptr, 0, []int16{2, 2, 1, 2})
	b.SetI32(sections+8+4, fromDesc(animSectioned))
	b.U8(1, trackRawPos).I16(0).I16(half3, 0, 0)
	b.SetI32(sections+24+4, fromDesc(animSectioned))
	b.U8(1, trackRawPos).I16(0).I16(half4, 0, 0)

	// ani: bone 1 is at (8, 0, 0), in block 1 of the .ani file
	b.SetI32(desc(animInAni)+16, 2)
	b.SetI32(desc(animInAni)+52, 1)
	b.SetI32(348, int32(str("test.ani")))
	b.SetI32(352, 2)
	b.SetI32(356, int32(b.Pos()))
	b.I32(0, 0, 16, 26)
	ani := (&testutil.Builder{}).U8('I', 'D', 'A', 'G').Zero(12)
	ani.U8(1, trackRawPos).I16(0).I16(half8, 0, 0)

	// One sequence blending walk and delta, with every kind of detail
	seq := b.Pos()
	b.Zero(seqDesc)
	b.SetI32(188, 1)
	b.SetI32(192, int32(seq))
	fromSeq := func() int32 { return int32(b.Pos() - seq) }
	b.SetI32(seq+4, int32(str("walk")-seq))
	b.SetI32(seq+8, int32(str("ACT_WALK")-seq))
	b.SetI32(seq+24, 1)
	b.SetI32(seq+28, fromSeq())
	event := b.Pos()
	b.F32(0.5).I32(0, eventNewStyle).Zero(64 + 4)
	copy(b.Bytes[event+12:], "left")
	b.SetI32(event+76, int32(str("AE_FOOTSTEP")-event))
	b.SetI32(seq+68, 2)
	b.SetI32(seq+72, 1)
	b.SetI32(seq+60, fromSeq())
	b.I16(animWalk, animDelta)
	b.SetI32(seq+160, fromSeq())
	b.F32(0, 1, 0)
	b.SetI32(seq+148, 1)
	b.SetI32(seq+152, fromSeq())
	b.I16(0, -1).I32(0).F32(0, 0.1, 0.9, 1)
	b.SetI32(seq+156, fromSeq())
	b.F32(1, 0.5)
	b.SetI32(seq+164, 1)
	b.SetI32(seq+168, fromSeq())
	b.I32(0).F32(1, 0).Zero(20)
	b.SetI32(seq+172, int32(str("seqkv")-seq))
	b.SetI32(seq+176, 6)
	b.SetI32(seq+188, 1)
	b.SetI32(seq+184, fromSeq())
	modifier := b.Pos()
	b.I32(0)
	b.SetI32(modifier, int32(str("armed")-modifier))

	// Two transition nodes
	b.SetI32(248, 2)
	b.SetI32(252, int32(b.Pos()))
	b.U8(0, 2, 1, 0)
	names := b.Pos()
	b.Zero(8)
	b.SetI32(256, int32(names))
	b.SetI32(names, int32(str("stand")))
	b.SetI32(names+4, int32(str("crouch")))

	b.SetI32(76, int32(len(b.Bytes)))
	return b.Bytes, ani.Bytes
}

func readAnimatedMdl(t *testing.T) (*Mdl, []byte) {
	t.Helper()
	mdlFile, aniFile := buildAnimatedMdl()
	m, err := ReadFromStream(bytes.NewReader(mdlFile))
	if err != nil {
		t.Fatal(err)
	}
	return m, aniFile
}

func approxQuat(a, b mgl32.Quat) bool {
	return math.Abs(float64(a.Dot(b))) > 1-1e-6
}

func TestAnimationFrame(t *testing.T) {
	m, _ := readAnimatedMdl(t)

	for frame, want := range []struct{ rotX, posX, posY float32 }{{10, 4, 7}, {20, 6, 9}, {30, 6, 9}} {
		poses, err := m.AnimationFrame(animWalk, frame)
		if err != nil {
			t.Fatal(err)
		}
		if !approxQuat(poses[0].Rotation, mgl32.Quat{W: 0.8, V: mgl32.Vec3{0, 0, 0.6}}) || poses[0].Position != (mgl32.Vec3{1.5, -2, 0.25}) {
			t.Errorf("walk frame %d bone 0: got %+v", frame, poses[0])
		}
		// Compressed values are scaled, then added to the bone's default
		wantRot := angleQuaternion(mgl32.Vec3{want.rotX * 0.01, 0, 5 * 0.01})
		wantPos := mgl32.Vec3{want.posX*0.5 + 1, want.posY*0.5 + 2, 3}
		if !approxQuat(poses[1].Rotation, wantRot) || !poses[1].Position.ApproxEqual(wantPos) {
			t.Errorf("walk frame %d bone 1: got %+v, want rotation %v position %v", frame, poses[1], wantRot, wantPos)
		}
	}

	delta, err := m.AnimationFrame(animDelta, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !approxQuat(delta[0].Rotation, mgl32.QuatIdent()) || delta[0].Position != (mgl32.Vec3{}) || delta[1].Position != (mgl32.Vec3{}) {
		t.Errorf("delta: got %+v, want identities", delta)
	}
	if quarterTurn := mgl32.QuatRotate(math.Pi/2, mgl32.Vec3{0, 0, 1}); math.Abs(float64(delta[1].Rotation.Dot(quarterTurn))) < 1-1e-4 {
		t.Errorf("delta bone 1: got rotation %v, want %v", delta[1].Rotation, quarterTurn)
	}

	idle, err := m.AnimationFrame(animIdle, 0)
	if err != nil {
		t.Fatal(err)
	}
	for bone, pose := range idle {
		if pose.Position != m.Bones[bone].Position || pose.Rotation != m.Bones[bone].Quaternion.Quat() {
			t.Errorf("idle bone %d: got %+v, want its default pose", bone, pose)
		}
	}

	for frame, want := range []float32{1.5, 2, 3, 4} {
		poses, err := m.AnimationFrame(animSectioned, frame)
		if err != nil {
			t.Fatal(err)
		}
		if poses[1].Position.X() != want {
			t.Errorf("sectioned frame %d: got bone 1 x %v, want %v", frame, poses[1].Position.X(), want)
		}
	}

	for name, c := range map[string][2]int{"animation past end": {numAnims, 0}, "frame past end": {animWalk, 3}, "negative frame": {animWalk, -1}} {
		if _, err := m.AnimationFrame(c[0], c[1]); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestLoadAnimBlocks(t *testing.T) {
	m, ani := readAnimatedMdl(t)
	if m.AnimBlockName != "test.ani" || !slices.Equal(m.AnimBlocks, []AnimBlock{{0, 0}, {16, 26}}) {
		t.Errorf("animation blocks: got %q %+v", m.AnimBlockName, m.AnimBlocks)
	}

	if _, err := m.AnimationFrame(animInAni, 0); err == nil {
		t.Error("before loading the .ani file: expected an error")
	}
	if err := m.LoadAnimBlocks(bytes.NewReader([]byte("nope"))); err == nil {
		t.Error("loading a file that isn't an .ani: expected an error")
	}
	if err := m.LoadAnimBlocks(bytes.NewReader(ani[:20])); err == nil {
		t.Error("loading a truncated .ani: expected an error")
	}
	if err := m.LoadAnimBlocks(bytes.NewReader(ani)); err != nil {
		t.Fatal(err)
	}
	poses, err := m.AnimationFrame(animInAni, 1)
	if err != nil {
		t.Fatal(err)
	}
	if poses[1].Position != (mgl32.Vec3{8, 0, 0}) {
		t.Errorf("bone 1: got %v, want (8, 0, 0)", poses[1].Position)
	}
}

func TestAnimationDetails(t *testing.T) {
	m, _ := readAnimatedMdl(t)
	if !slices.Equal(m.AnimDescNames, []string{"@walk", "@delta", "@idle", "@sectioned", "@ani"}) {
		t.Errorf("animation names: got %q", m.AnimDescNames)
	}
	want := []Movement{{EndFrame: 2, MotionFlags: 1, V1: 10, Vector: mgl32.Vec3{20, 0, 0}, Position: mgl32.Vec3{20, 0, 0}}}
	if !slices.Equal(m.AnimMovements[animWalk], want) {
		t.Errorf("movements: got %+v, want %+v", m.AnimMovements[animWalk], want)
	}
	if rules := m.AnimIKRules[animWalk]; len(rules) != 1 || rules[0].Bone != 1 || rules[0].Start != 0.1 || rules[0].End != 0.9 || m.AnimIKRuleAttachments[animWalk][0] != "foot" {
		t.Errorf("IK rules: got %+v attached to %q", rules, m.AnimIKRuleAttachments[animWalk])
	}
}

func TestSequences(t *testing.T) {
	m, _ := readAnimatedMdl(t)
	if !slices.Equal(m.SequenceLabels, []string{"walk"}) || !slices.Equal(m.SequenceActivityNames, []string{"ACT_WALK"}) {
		t.Errorf("sequence names: got %q, %q", m.SequenceLabels, m.SequenceActivityNames)
	}
	want := SequenceData{
		Events:            []SequenceEvent{{Cycle: 0.5, Type: eventNewStyle, Options: "left", Name: "AE_FOOTSTEP"}},
		Animations:        []int16{animWalk, animDelta},
		PoseKeys:          []float32{0, 1, 0},
		AutoLayers:        []AutoLayer{{Sequence: 0, Pose: -1, Peak: 0.1, Tail: 0.9, End: 1}},
		BoneWeights:       []float32{1, 0.5},
		IKLocks:           []IKLock{{PosWeight: 1}},
		KeyValues:         "seqkv",
		ActivityModifiers: []string{"armed"},
	}
	if len(m.Sequences) != 1 || !reflect.DeepEqual(m.Sequences[0], want) {
		t.Errorf("got %+v, want %+v", m.Sequences, want)
	}
}

func TestTransitions(t *testing.T) {
	m, _ := readAnimatedMdl(t)
	if !slices.Equal(m.NodeNames, []string{"stand", "crouch"}) || !reflect.DeepEqual(m.Transitions, [][]uint8{{0, 2}, {1, 0}}) {
		t.Errorf("got nodes %q, transitions %v", m.NodeNames, m.Transitions)
	}
}

func TestAnimationFrameRejectsCorruptData(t *testing.T) {
	m, _ := readAnimatedMdl(t)
	// A channel whose spans end before the frame
	walk := &m.AnimDescs[animWalk]
	start := int64(m.Header.LocalAnimationOffset) + int64(walk.AnimIndex)
	bone1 := start + int64(int16(uint16(m.buf[start+2])|uint16(m.buf[start+3])<<8))
	rotX := bone1 + 4 + int64(int16(uint16(m.buf[bone1+4])|uint16(m.buf[bone1+5])<<8))
	m.buf[rotX+1] = 0
	if _, err := m.AnimationFrame(animWalk, 2); err == nil {
		t.Error("values ending early: expected an error")
	}
}

func TestAngleQuaternion(t *testing.T) {
	for _, c := range []struct {
		angles mgl32.Vec3
		axis   mgl32.Vec3
	}{
		{mgl32.Vec3{math.Pi / 2, 0, 0}, mgl32.Vec3{1, 0, 0}},
		{mgl32.Vec3{0, math.Pi / 2, 0}, mgl32.Vec3{0, 1, 0}},
		{mgl32.Vec3{0, 0, math.Pi / 2}, mgl32.Vec3{0, 0, 1}},
	} {
		if got, want := angleQuaternion(c.angles), mgl32.QuatRotate(math.Pi/2, c.axis); !approxQuat(got, want) {
			t.Errorf("%v: got %v, want %v", c.angles, got, want)
		}
	}
}

func TestFloat16(t *testing.T) {
	for h, want := range map[uint16]float32{0x3c00: 1, 0xc000: -2, 0x3400: 0.25, 0x0001: 1.0 / 1024 / 16384, 0x7bff: 65504, 0: 0} {
		if got := float16(h); got != want {
			t.Errorf("%#04x: got %v, want %v", h, got, want)
		}
	}
}
