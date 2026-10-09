package mdl

import (
	"bytes"
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

// buildFaceMdl builds buildMdl's model with three flex descriptions, two flex controllers and their UI, two flex
// rules, a mouth, an eyeball, two mesh flexes (the second with wrinkles) and a bone flex driver. With fixedPoint, the
// vertex animation deltas are fixed point instead of half-precision floats.
func buildFaceMdl(fixedPoint bool) []byte {
	b := buildMdl()
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
	// names writes an array of name offsets, each relative to itself
	names := func(values ...string) int {
		at := b.Pos()
		b.Zero(4 * len(values))
		for i, v := range values {
			b.SetI32(at+4*i, int32(str(v)-(at+4*i)))
		}
		return at
	}

	b.SetI32(260, 3)
	b.SetI32(264, int32(names("smile_l", "smile_r", "jaw_drop")))

	controllers := b.Pos()
	b.Zero(2 * 20)
	b.SetI32(268, 2)
	b.SetI32(272, int32(controllers))
	for i, c := range []struct {
		kind, name string
		min, max   float32
	}{{"default", "smile", 0, 1}, {"jaw", "jaw", -1, 1}} {
		at := controllers + i*20
		b.SetI32(at, int32(str(c.kind)-at))
		b.SetI32(at+4, int32(str(c.name)-at))
		b.SetI32(at+8, -1)
		set(at+12, c.min, c.max)
	}

	// smile_l = smile * 0.5; jaw_drop = smile_l + jaw
	rules := b.Pos()
	b.Zero(2 * 12)
	b.SetI32(276, 2)
	b.SetI32(280, int32(rules))
	for i, rule := range []struct {
		flex int32
		ops  [][2]int32
	}{
		{0, [][2]int32{{FlexOpFetch1, 0}, {FlexOpConst, int32(math.Float32bits(0.5))}, {FlexOpMul, 0}}},
		{2, [][2]int32{{FlexOpFetch2, 0}, {FlexOpFetch1, 1}, {FlexOpAdd, 0}}},
	} {
		at := rules + i*12
		b.SetI32(at, rule.flex)
		b.SetI32(at+4, int32(len(rule.ops)))
		b.SetI32(at+8, int32(b.Pos()-at))
		for _, op := range rule.ops {
			b.I32(op[0], op[1])
		}
	}

	// A stereo control over both controllers, and a mono one over the jaw; controllers are offsets to them
	ui := b.Pos()
	b.Zero(2 * 20)
	b.SetI32(384, 2)
	b.SetI32(388, int32(ui))
	b.SetI32(ui, int32(str("smile")-ui))
	b.SetI32(ui+4, int32(controllers-ui))
	b.SetI32(ui+8, int32(controllers+20-ui))
	b.Bytes[ui+17] = 1
	b.SetI32(ui+20, int32(str("jaw")-(ui+20)))
	b.SetI32(ui+24, int32(controllers+20-(ui+20)))

	b.SetI32(292, 1)
	b.SetI32(296, int32(b.Pos()))
	b.I32(1).F32(0, 1, 0).I32(2)

	// An eyeball on model 0
	eyeball := b.Pos()
	b.Zero(172)
	b.SetI32(modelsAt+100, 1)
	b.SetI32(modelsAt+104, int32(eyeball-modelsAt))
	b.SetI32(eyeball, int32(str("eye_r")-eyeball))
	b.SetI32(eyeball+4, 1)
	set(eyeball+8, 1, 2, 3)
	set(eyeball+24, 0.5)
	b.SetI32(eyeball+116, 0)

	// Two flexes on model 0's mesh, which has 3 vertices
	b.SetI32(meshAt+8, 3)
	flexes := b.Pos()
	b.Zero(2 * 60)
	b.SetI32(meshAt+16, 2)
	b.SetI32(meshAt+20, int32(flexes-meshAt))
	for i, flex := range []struct{ desc, pair, kind int32 }{{0, 1, 0}, {2, -1, 1}} {
		at := flexes + i*60
		b.SetI32(at, flex.desc)
		set(at+4, 0, 1, 1, 1)
		b.SetI32(at+28, flex.pair)
		b.Bytes[at+32] = byte(flex.kind)
	}
	b.SetI32(flexes+20, 2)
	b.SetI32(flexes+24, int32(b.Pos()-flexes))
	if fixedPoint {
		b.SetI32(152, StudiohdrFlagsVertAnimFixedPointScale)
		b.SetF32(392, 0.5)
		b.U8(1, 0, 255, 128).I16(3, -4, 1, 0, 0, 2) // deltas (1.5, -2, 0.5), (0, 0, 1)
		b.U8(2, 0, 0, 0).I16(6, 0, 0, 0, 0, 0)
	} else {
		b.U8(1, 0, 255, 128).I16(half1_5, halfNeg2, 0x3800, 0, 0, 0x3c00) // deltas (1.5, -2, 0.5), (0, 0, 1)
		b.U8(2, 0, 0, 0).I16(half3, 0, 0, 0, 0, 0)
	}
	b.SetI32(flexes+60+20, 1)
	b.SetI32(flexes+60+24, int32(b.Pos()-(flexes+60)))
	b.U8(0, 0, 0, 0).I16(0, 0, 0, 0, 0, 0, 100) // a wrinkle vertex animation

	// A bone flex driver: bone 1's z drives the jaw
	driver := b.Pos()
	b.SetI32(hdr2At+24, 1)
	b.SetI32(hdr2At+28, int32(driver-hdr2At))
	b.I32(1, 1, 24).Zero(12)
	b.I32(2, 1).F32(0, 10)

	b.SetI32(76, int32(len(b.Bytes)))
	return b.Bytes
}

func TestReadFaces(t *testing.T) {
	for name, fixedPoint := range map[string]bool{"half floats": false, "fixed point": true} {
		t.Run(name, func(t *testing.T) {
			m, err := ReadFromStream(bytes.NewReader(buildFaceMdl(fixedPoint)))
			if err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(m.FlexDescs, []string{"smile_l", "smile_r", "jaw_drop"}) {
				t.Errorf("flex descriptions: got %q", m.FlexDescs)
			}
			if !slices.Equal(m.FlexControllerNames, []string{"smile", "jaw"}) || !slices.Equal(m.FlexControllerTypes, []string{"default", "jaw"}) ||
				m.FlexControllers[1].Min != -1 || m.FlexControllers[1].Max != 1 {
				t.Errorf("flex controllers: got %+v named %q of types %q", m.FlexControllers, m.FlexControllerNames, m.FlexControllerTypes)
			}
			if len(m.FlexRules) != 2 || m.FlexRules[0].Ops[1] != (FlexOp{Op: FlexOpConst, Index: int32(math.Float32bits(0.5)), Value: 0.5}) {
				t.Errorf("flex rules: got %+v", m.FlexRules)
			}
			wantUI := []FlexControllerUI{
				{Name: "smile", Controller: -1, Left: 0, Right: 1, Value: -1, Stereo: true},
				{Name: "jaw", Controller: 1, Left: -1, Right: -1, Value: -1},
			}
			if !reflect.DeepEqual(m.FlexControllerUI, wantUI) {
				t.Errorf("flex controller UI: got %+v, want %+v", m.FlexControllerUI, wantUI)
			}
			if want := []Mouth{{Bone: 1, Forward: mgl32.Vec3{0, 1, 0}, FlexDesc: 2}}; !slices.Equal(m.Mouths, want) {
				t.Errorf("mouths: got %+v", m.Mouths)
			}

			model := m.BodyParts[0].Models[0]
			if len(model.Eyeballs) != 1 || model.EyeballNames[0] != "eye_r" || model.Eyeballs[0].Bone != 1 || model.Eyeballs[0].Org != (mgl32.Vec3{1, 2, 3}) || model.Eyeballs[0].Radius != 0.5 {
				t.Errorf("eyeballs: got %+v named %q", model.Eyeballs, model.EyeballNames)
			}

			if len(model.Flexes) != 1 || len(model.Flexes[0]) != 2 {
				t.Fatalf("mesh flexes: got %+v", model.Flexes)
			}
			flex := model.Flexes[0][0]
			wantAnims := []VertAnim{
				{Index: 1, Speed: 255, Side: 128, Delta: mgl32.Vec3{1.5, -2, 0.5}, NormalDelta: mgl32.Vec3{0, 0, 1}},
				{Index: 2, Delta: mgl32.Vec3{3, 0, 0}},
			}
			if flex.Header.FlexDesc != 0 || flex.Header.FlexPair != 1 || !slices.Equal(flex.VertAnims, wantAnims) {
				t.Errorf("flex 0: got %+v, want vertex animations %+v", flex, wantAnims)
			}
			if wrinkle := model.Flexes[0][1]; len(wrinkle.VertAnims) != 1 || wrinkle.VertAnims[0].WrinkleDelta != 100 {
				t.Errorf("wrinkle flex: got %+v", wrinkle)
			}

			wantDrivers := []BoneFlexDriver{{Bone: 1, Controls: []BoneFlexDriverControl{{BoneComponent: 2, FlexController: 1, Max: 10}}}}
			if !reflect.DeepEqual(m.BoneFlexDrivers, wantDrivers) {
				t.Errorf("bone flex drivers: got %+v", m.BoneFlexDrivers)
			}

			weights, err := m.FlexWeights([]float32{0.8, 0.3})
			if err != nil {
				t.Fatal(err)
			}
			if want := []float32{0.4, 0, 0.7}; !slices.EqualFunc(weights, want, func(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-6 }) {
				t.Errorf("flex weights: got %v, want %v", weights, want)
			}
		})
	}
}

func TestFlexWeightsOps(t *testing.T) {
	constant := func(v float32) FlexOp { return FlexOp{Op: FlexOpConst, Value: v} }
	op := func(code, index int32) FlexOp { return FlexOp{Op: code, Index: index} }
	controllers := []float32{0.5, -0.5, 2}

	for name, c := range map[string]struct {
		ops  []FlexOp
		want float32
	}{
		"fetch controller":     {[]FlexOp{op(FlexOpFetch1, 2)}, 2},
		"fetch weight":         {[]FlexOp{op(FlexOpFetch2, 0)}, 0.25},
		"subtract":             {[]FlexOp{constant(5), constant(3), op(FlexOpSub, 0)}, 2},
		"divide":               {[]FlexOp{constant(3), constant(2), op(FlexOpDiv, 0)}, 1.5},
		"divide by near zero":  {[]FlexOp{constant(3), constant(0), op(FlexOpDiv, 0)}, 0},
		"negate":               {[]FlexOp{constant(3), op(FlexOpNeg, 0)}, -3},
		"max":                  {[]FlexOp{constant(3), constant(4), op(FlexOpMax, 0)}, 4},
		"min":                  {[]FlexOp{constant(3), constant(4), op(FlexOpMin, 0)}, 3},
		"two-way, low half":    {[]FlexOp{op(FlexOp2Way0, 1)}, 0.5},
		"two-way, high half":   {[]FlexOp{op(FlexOp2Way1, 0)}, 0.5},
		"combo":                {[]FlexOp{constant(2), constant(3), constant(4), op(FlexOpCombo, 3)}, 24},
		"dominate":             {[]FlexOp{constant(2), constant(0.5), constant(0.5), op(FlexOpDominate, 2)}, 1.5},
		"n-way, inside ramp":   {[]FlexOp{constant(0), constant(1), constant(3), constant(4), constant(2), op(FlexOpNWay, 0)}, 0.5},
		"n-way, rising":        {[]FlexOp{constant(1.5), constant(2.5), constant(3), constant(4), constant(2), op(FlexOpNWay, 0)}, 0.25},
		"n-way, outside":       {[]FlexOp{constant(3), constant(4), constant(5), constant(6), constant(2), op(FlexOpNWay, 0)}, 0},
		"parser-only op skips": {[]FlexOp{constant(7), op(10, 0)}, 7},
	} {
		t.Run(name, func(t *testing.T) {
			m := &Mdl{
				FlexDescs:       make([]string, 2),
				FlexControllers: make([]FlexController, len(controllers)),
				FlexRules:       []FlexRule{{Flex: 0, Ops: []FlexOp{constant(0.25)}}, {Flex: 1, Ops: c.ops}},
			}
			weights, err := m.FlexWeights(controllers)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(float64(weights[1]-c.want)) > 1e-6 {
				t.Errorf("got %v, want %v", weights[1], c.want)
			}
		})
	}

	for name, ops := range map[string][]FlexOp{
		"stack underflow":       {constant(1), op(FlexOpAdd, 0)},
		"controller past end":   {op(FlexOpFetch1, 3)},
		"weight past end":       {op(FlexOpFetch2, 2)},
		"unsupported eyelid op": {constant(1), constant(1), constant(1), op(FlexOpDMEUpperLid, 0)},
		"no value":              {},
	} {
		m := &Mdl{FlexDescs: make([]string, 2), FlexControllers: make([]FlexController, len(controllers)), FlexRules: []FlexRule{{Flex: 1, Ops: ops}}}
		if _, err := m.FlexWeights(controllers); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	m := &Mdl{FlexControllers: make([]FlexController, 2)}
	if _, err := m.FlexWeights([]float32{1}); err == nil {
		t.Error("wrong number of controller values: expected an error")
	}
}
