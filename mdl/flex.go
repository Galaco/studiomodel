package mdl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/galaco/studiomodel/internal"
	"github.com/go-gl/mathgl/mgl32"
)

// StudiohdrFlagsVertAnimFixedPointScale means vertex animation deltas are fixed point, scaled by
// Studiohdr.VertAnimFixedPointScale, rather than half-precision floats
const StudiohdrFlagsVertAnimFixedPointScale = 1 << 21

// FlexController is a value that drives facial expressions, such as how far the jaw is open
// Corresponds to mstudioflexcontroller_t in studio.h
type FlexController struct {
	// TypeIndex - offset to the controller's type name (relative to this struct)
	TypeIndex int32
	// NameIndex - offset to the controller's name (relative to this struct)
	NameIndex int32
	// LocalToGlobal is set by the engine at load time
	LocalToGlobal int32
	// Min
	Min float32
	// Max
	Max float32
}

// flexRule is mstudioflexrule_t in studio.h
type flexRule struct {
	Flex    int32
	NumOps  int32
	OpIndex int32
}

// FlexRule computes the weight of one flex description from the flex controllers, as a stack program
type FlexRule struct {
	// Flex - the flex description whose weight this sets
	Flex int32
	// Ops
	Ops []FlexOp
}

// FlexOp is one step of a FlexRule. Index and Value are the same four bytes; which one an op uses depends on Op.
// Corresponds to mstudioflexop_t in studio.h
type FlexOp struct {
	Op    int32
	Index int32
	Value float32
}

// FlexOp.Op values
const (
	FlexOpConst       = 1 // push Value
	FlexOpFetch1      = 2 // push the value of flex controller Index
	FlexOpFetch2      = 3 // push the weight already computed for flex description Index
	FlexOpAdd         = 4
	FlexOpSub         = 5
	FlexOpMul         = 6
	FlexOpDiv         = 7
	FlexOpNeg         = 8
	FlexOpMax         = 13
	FlexOpMin         = 14
	FlexOp2Way0       = 15 // push flex controller Index remapped from -1..0 to 1..0
	FlexOp2Way1       = 16 // push flex controller Index remapped from 0..1 to 0..1
	FlexOpNWay        = 17
	FlexOpCombo       = 18 // multiply the top Index values together
	FlexOpDominate    = 19
	FlexOpDMELowerLid = 20
	FlexOpDMEUpperLid = 21

	// flexRuleStackDepth is the engine's limit
	flexRuleStackDepth = 32
)

// flexOp is mstudioflexop_t in studio.h
type flexOp struct {
	Op    int32
	Value uint32
}

// flexControllerUI is mstudioflexcontrollerui_t in studio.h
type flexControllerUI struct {
	NameIndex int32
	Index0    int32
	Index1    int32
	Index2    int32
	RemapType uint8
	Stereo    uint8
	_         [2]uint8
}

// FlexControllerUI groups flex controllers into one control in a face posing tool. Unused controllers are -1.
type FlexControllerUI struct {
	// Name
	Name string
	// Controller is the flex controller of a mono control, and Left and Right those of a stereo one
	Controller, Left, Right int
	// Value - the controller that selects between this control's shapes, for an n-way control
	Value int
	// RemapType - how the control maps to its controllers: 0 pass-through, 1 two-way, 2 n-way, 3 eyelid
	RemapType uint8
	// Stereo
	Stereo bool
}

// Mouth is a bone and flex description that move with speech
// Corresponds to mstudiomouth_t in studio.h
type Mouth struct {
	// Bone
	Bone int32
	// Forward
	Forward mgl32.Vec3
	// FlexDesc
	FlexDesc int32
}

// Eyeball
// Corresponds to mstudioeyeball_t in studio.h
type Eyeball struct {
	// NameIndex - offset to the eyeball's name (relative to this struct)
	NameIndex int32
	// Bone
	Bone int32
	// Org - centre of the eyeball, relative to its bone
	Org mgl32.Vec3
	// ZOffset
	ZOffset float32
	// Radius
	Radius float32
	// Up
	Up mgl32.Vec3
	// Forward
	Forward mgl32.Vec3
	// Texture - not set in practice. The eyeball's mesh is the one whose MaterialType is 1 and whose MaterialParam is
	// this eyeball's index.
	Texture int32
	_       int32
	// IrisScale
	IrisScale float32
	_         int32
	// UpperFlexDesc - flex descriptions that raise, neutralise and lower the upper lid
	UpperFlexDesc [3]int32
	// LowerFlexDesc - flex descriptions that raise, neutralise and lower the lower lid
	LowerFlexDesc [3]int32
	// UpperTarget - upper lid angles, in radians, for raised, neutral and lowered
	UpperTarget [3]float32
	// LowerTarget - lower lid angles, in radians, for raised, neutral and lowered
	LowerTarget [3]float32
	// UpperLidFlexDesc - the flex description the upper lid's flexes follow
	UpperLidFlexDesc int32
	// LowerLidFlexDesc - the flex description the lower lid's flexes follow
	LowerLidFlexDesc int32
	_                [4]int32
	// NonFACS
	NonFACS uint8
	_       [3]uint8
	_       [7]int32
}

// Flex is one shape a mesh can blend towards, such as a smile
// Corresponds to mstudioflex_t in studio.h
type Flex struct {
	// FlexDesc - the flex description whose weight drives this flex
	FlexDesc int32
	// Target0 to Target3 shape how the weight applies: none below Target0, rising to full at Target1, full to
	// Target2, falling to none at Target3
	Target0 float32
	Target1 float32
	Target2 float32
	Target3 float32
	// NumVerts
	NumVerts int32
	// VertIndex - byte offset from start of this struct to its vertex animations
	VertIndex int32
	// FlexPair - the flex description for the other side, for stereo flexes
	FlexPair int32
	// VertAnimType - 0 for normal vertex animations, 1 for ones with a wrinkle
	VertAnimType uint8
	_            [3]uint8
	_            [6]int32
}

// FlexData contains a parsed flex with its vertex animations
type FlexData struct {
	Header Flex
	// VertAnims
	VertAnims []VertAnim
}

// vertAnim is mstudiovertanim_t in studio.h
type vertAnim struct {
	Index  uint16
	Speed  uint8
	Side   uint8
	Delta  [3]uint16
	NDelta [3]uint16
}

// VertAnim moves one vertex of a mesh, at the flex's full weight
type VertAnim struct {
	// Index of the vertex within its mesh
	Index uint16
	// Speed
	Speed uint8
	// Side - for stereo flexes, how far the vertex follows the left (255) or right (0) weight
	Side uint8
	// Delta - change in position
	Delta mgl32.Vec3
	// NormalDelta - change in normal
	NormalDelta mgl32.Vec3
	// WrinkleDelta - for wrinkle flexes, the change in wrinkle map weight, as stored
	WrinkleDelta int16
}

// boneFlexDriver is mstudioboneflexdriver_t in studio.h
type boneFlexDriver struct {
	BoneIndex    int32
	ControlCount int32
	ControlIndex int32
	_            [3]int32
}

// BoneFlexDriver drives flex controllers from a bone's position
type BoneFlexDriver struct {
	// Bone
	Bone int32
	// Controls
	Controls []BoneFlexDriverControl
}

// BoneFlexDriverControl maps one component of a bone's position onto a flex controller
// Corresponds to mstudioboneflexdrivercontrol_t in studio.h
type BoneFlexDriverControl struct {
	// BoneComponent - 0, 1 or 2 for x, y or z
	BoneComponent int32
	// FlexController
	FlexController int32
	// Min - the position that maps to the controller's minimum
	Min float32
	// Max - the position that maps to the controller's maximum
	Max float32
}

// FlexWeights runs the flex rules, as the engine does, to turn a value for each flex controller into a weight for
// each flex description. Flex descriptions without a rule get 0.
func (mdl *Mdl) FlexWeights(controllers []float32) ([]float32, error) {
	if len(controllers) != len(mdl.FlexControllers) {
		return nil, fmt.Errorf("got %d controller values for %d flex controllers", len(controllers), len(mdl.FlexControllers))
	}
	weights := make([]float32, len(mdl.FlexDescs))

	controller := func(index int32) (float32, error) {
		if index < 0 || int(index) >= len(controllers) {
			return 0, fmt.Errorf("flex controller %d out of range", index)
		}
		return controllers[index], nil
	}

	for r, rule := range mdl.FlexRules {
		fail := func(err error) ([]float32, error) {
			return nil, fmt.Errorf("flex rule %d: %w", r, err)
		}
		if rule.Flex < 0 || int(rule.Flex) >= len(weights) {
			return fail(fmt.Errorf("sets flex description %d of %d", rule.Flex, len(weights)))
		}

		stack := make([]float32, 0, flexRuleStackDepth)
		for _, op := range rule.Ops {
			n := len(stack)
			// need checks the stack holds enough values for the op
			need := func(count int) error {
				if count < 1 || n < count {
					return fmt.Errorf("op %d needs %d values, stack has %d", op.Op, count, n)
				}
				return nil
			}

			switch op.Op {
			case FlexOpConst:
				stack = append(stack, op.Value)
			case FlexOpFetch1, FlexOp2Way0, FlexOp2Way1:
				v, err := controller(op.Index)
				if err != nil {
					return fail(err)
				}
				switch op.Op {
				case FlexOp2Way0:
					v = remapClamped(v, -1, 0, 1, 0)
				case FlexOp2Way1:
					v = remapClamped(v, 0, 1, 0, 1)
				}
				stack = append(stack, v)
			case FlexOpFetch2:
				if op.Index < 0 || int(op.Index) >= len(weights) {
					return fail(fmt.Errorf("flex description %d out of range", op.Index))
				}
				stack = append(stack, weights[op.Index])
			case FlexOpAdd, FlexOpSub, FlexOpMul, FlexOpDiv, FlexOpMax, FlexOpMin:
				if err := need(2); err != nil {
					return fail(err)
				}
				a, b := stack[n-2], stack[n-1]
				switch op.Op {
				case FlexOpAdd:
					a += b
				case FlexOpSub:
					a -= b
				case FlexOpMul:
					a *= b
				case FlexOpDiv:
					if b > 0.0001 {
						a /= b
					} else {
						a = 0
					}
				case FlexOpMax:
					a = max(a, b)
				case FlexOpMin:
					a = min(a, b)
				}
				stack[n-2] = a
				stack = stack[:n-1]
			case FlexOpNeg:
				if err := need(1); err != nil {
					return fail(err)
				}
				stack[n-1] = -stack[n-1]
			case FlexOpNWay:
				// The top value names the controller that selects a shape, and the four below it are the ramp of
				// selector values over which this shape fades in, holds and fades out
				if err := need(5); err != nil {
					return fail(err)
				}
				selector, err := controller(int32(stack[n-1]))
				if err != nil {
					return fail(err)
				}
				value, err := controller(op.Index)
				if err != nil {
					return fail(err)
				}
				x, y, z, w := stack[n-5], stack[n-4], stack[n-3], stack[n-2]
				switch {
				case selector <= x || selector >= w:
					selector = 0
				case selector < y:
					selector = remapClamped(selector, x, y, 0, 1)
				case selector > z:
					selector = remapClamped(selector, z, w, 1, 0)
				default:
					selector = 1
				}
				stack[n-5] = selector * value
				stack = stack[:n-4]
			case FlexOpCombo:
				// Multiply the top Index values together
				if err := need(int(op.Index)); err != nil {
					return fail(err)
				}
				first := n - int(op.Index)
				for i := first + 1; i < n; i++ {
					stack[first] *= stack[i]
				}
				stack = stack[:first+1]
			case FlexOpDominate:
				// Scale the value below the top Index values by one minus their product
				if err := need(int(op.Index) + 1); err != nil {
					return fail(err)
				}
				first := n - int(op.Index)
				dominance := stack[first]
				for i := first + 1; i < n; i++ {
					dominance *= stack[i]
				}
				stack[first-1] *= 1 - dominance
				stack = stack[:first]
			case FlexOpDMELowerLid, FlexOpDMEUpperLid:
				return fail(fmt.Errorf("op %d is not supported", op.Op))
			}
			// Other ops, such as the parser's own open and close brackets, do nothing in the engine either

			if len(stack) > flexRuleStackDepth {
				return fail(errors.New("stack overflow"))
			}
		}
		if len(stack) == 0 {
			return fail(errors.New("leaves no value"))
		}
		weights[rule.Flex] = stack[0]
	}
	return weights, nil
}

// remapClamped maps v from [a, b] to [c, d], clamping to [c, d]
func remapClamped(v, a, b, c, d float32) float32 {
	if a == b {
		if v >= b {
			return d
		}
		return c
	}
	t := min(max((v-a)/(b-a), 0), 1)
	return c + (d-c)*t
}

// readFlexes reads the flex descriptions, controllers, rules, controller UI and mouths
func readFlexes(buf []byte, header *Studiohdr, out *Mdl) error {
	descs, err := internal.ReadSlice[int32](buf, int64(header.FlexDescIndex), int64(header.FlexDescCount), "flex descriptions")
	if err != nil {
		return err
	}
	// Each flex description is just a name offset, relative to itself
	if out.FlexDescs, err = readNames(buf, int64(header.FlexDescIndex), descs, func(nameIndex int32) int32 { return nameIndex }, "flex description"); err != nil {
		return err
	}

	controllersStart := int64(header.FlexControllerIndex)
	if out.FlexControllers, err = internal.ReadSlice[FlexController](buf, controllersStart, int64(header.FlexControllerCount), "flex controllers"); err != nil {
		return err
	}
	if out.FlexControllerNames, err = readNames(buf, controllersStart, out.FlexControllers, func(c FlexController) int32 { return c.NameIndex }, "flex controller"); err != nil {
		return err
	}
	if out.FlexControllerTypes, err = readNames(buf, controllersStart, out.FlexControllers, func(c FlexController) int32 { return c.TypeIndex }, "flex controller type"); err != nil {
		return err
	}

	rules, err := internal.ReadSlice[flexRule](buf, int64(header.FlexRulesIndex), int64(header.FlexRulesCount), "flex rules")
	if err != nil {
		return err
	}
	ruleSize := int64(binary.Size(flexRule{}))
	out.FlexRules = make([]FlexRule, len(rules))
	for i, rule := range rules {
		// OpIndex is relative to the start of this rule
		ops, err := internal.ReadSlice[flexOp](buf, int64(header.FlexRulesIndex)+int64(i)*ruleSize+int64(rule.OpIndex), int64(rule.NumOps), "flex ops")
		if err != nil {
			return fmt.Errorf("flex rule %d: %w", i, err)
		}
		out.FlexRules[i] = FlexRule{Flex: rule.Flex, Ops: make([]FlexOp, len(ops))}
		for j, op := range ops {
			out.FlexRules[i].Ops[j] = FlexOp{Op: op.Op, Index: int32(op.Value), Value: math.Float32frombits(op.Value)}
		}
	}

	if out.FlexControllerUI, err = readFlexControllerUI(buf, header); err != nil {
		return err
	}

	out.Mouths, err = internal.ReadSlice[Mouth](buf, int64(header.MouthsIndex), int64(header.MouthsCount), "mouths")
	return err
}

// readFlexControllerUI reads the flex controller UI, whose controllers are offsets to flex controllers, relative to
// the UI entry
func readFlexControllerUI(buf []byte, header *Studiohdr) ([]FlexControllerUI, error) {
	start := int64(header.FlexControllerUIIndex)
	entries, err := internal.ReadSlice[flexControllerUI](buf, start, int64(header.FlexControllerUICount), "flex controller UI")
	if err != nil {
		return nil, err
	}
	names, err := readNames(buf, start, entries, func(e flexControllerUI) int32 { return e.NameIndex }, "flex controller UI")
	if err != nil {
		return nil, err
	}

	entrySize := int64(binary.Size(flexControllerUI{}))
	controllerSize := int64(binary.Size(FlexController{}))
	out := make([]FlexControllerUI, len(entries))
	for i, e := range entries {
		entryPos := start + int64(i)*entrySize
		controller := func(offset int32) (int, error) {
			if offset == 0 {
				return -1, nil
			}
			at := entryPos + int64(offset) - int64(header.FlexControllerIndex)
			if at < 0 || at%controllerSize != 0 || at/controllerSize >= int64(header.FlexControllerCount) {
				return 0, fmt.Errorf("flex controller UI %d points at no flex controller", i)
			}
			return int(at / controllerSize), nil
		}
		var c [3]int
		for j, offset := range []int32{e.Index0, e.Index1, e.Index2} {
			if c[j], err = controller(offset); err != nil {
				return nil, err
			}
		}

		ui := FlexControllerUI{Name: names[i], Controller: -1, Left: -1, Right: -1, Value: -1, RemapType: e.RemapType, Stereo: e.Stereo != 0}
		if ui.Stereo {
			ui.Left, ui.Right = c[0], c[1]
		} else {
			ui.Controller = c[0]
		}
		if e.RemapType == 2 {
			ui.Value = c[2]
		}
		out[i] = ui
	}
	return out, nil
}

// readEyeballs reads a model's eyeballs and their names. The offset is relative to the model.
func readEyeballs(buf []byte, modelStart int64, model *Model) ([]Eyeball, []string, error) {
	start := modelStart + int64(model.EyeballIndex)
	eyeballs, err := internal.ReadSlice[Eyeball](buf, start, int64(model.NumEyeballs), "eyeballs")
	if err != nil {
		return nil, nil, err
	}
	names, err := readNames(buf, start, eyeballs, func(e Eyeball) int32 { return e.NameIndex }, "eyeball")
	return eyeballs, names, err
}

// readMeshFlexes reads a mesh's flexes and their vertex animations. Offsets are relative to the mesh, then the flex.
func readMeshFlexes(buf []byte, header *Studiohdr, meshStart int64, mesh *Mesh) ([]FlexData, error) {
	start := meshStart + int64(mesh.FlexIndex)
	flexes, err := internal.ReadSlice[Flex](buf, start, int64(mesh.NumFlexes), "flexes")
	if err != nil {
		return nil, err
	}

	fixedPoint := header.Flags&StudiohdrFlagsVertAnimFixedPointScale != 0
	component := func(v uint16) float32 {
		if fixedPoint {
			return float32(int16(v)) * header.VertAnimFixedPointScale
		}
		return float16(v)
	}
	vector := func(v [3]uint16) mgl32.Vec3 {
		return mgl32.Vec3{component(v[0]), component(v[1]), component(v[2])}
	}

	flexSize := int64(binary.Size(Flex{}))
	out := make([]FlexData, len(flexes))
	for i, flex := range flexes {
		// Wrinkle vertex animations carry two more bytes each
		stride := int64(binary.Size(vertAnim{}))
		if flex.VertAnimType == 1 {
			stride += 2
		}
		vertsStart := start + int64(i)*flexSize + int64(flex.VertIndex)
		raw, err := internal.ReadStrided[vertAnim](buf, vertsStart, int64(flex.NumVerts), stride, "vertex animations")
		if err != nil {
			return nil, fmt.Errorf("flex %d: %w", i, err)
		}
		anims := make([]VertAnim, len(raw))
		for j, v := range raw {
			anims[j] = VertAnim{Index: v.Index, Speed: v.Speed, Side: v.Side, Delta: vector(v.Delta), NormalDelta: vector(v.NDelta)}
			if flex.VertAnimType == 1 {
				at := vertsStart + int64(j)*stride + stride - 2
				anims[j].WrinkleDelta = int16(binary.LittleEndian.Uint16(buf[at:]))
			}
		}
		out[i] = FlexData{Header: flex, VertAnims: anims}
	}
	return out, nil
}

// readBoneFlexDrivers reads the bone flex drivers listed in the secondary header. Offsets are relative to the
// secondary header, then each driver.
func readBoneFlexDrivers(buf []byte, header *Studiohdr, header2 *Studiohdr2) ([]BoneFlexDriver, error) {
	if header2 == nil || header2.BoneFlexDriverCount == 0 {
		return nil, nil
	}
	start := int64(header.StudioHDR2Index) + int64(header2.BoneFlexDriverIndex)
	drivers, err := internal.ReadSlice[boneFlexDriver](buf, start, int64(header2.BoneFlexDriverCount), "bone flex drivers")
	if err != nil {
		return nil, err
	}
	driverSize := int64(binary.Size(boneFlexDriver{}))
	out := make([]BoneFlexDriver, len(drivers))
	for i, d := range drivers {
		controls, err := internal.ReadSlice[BoneFlexDriverControl](buf, start+int64(i)*driverSize+int64(d.ControlIndex), int64(d.ControlCount), "bone flex driver controls")
		if err != nil {
			return nil, fmt.Errorf("bone flex driver %d: %w", i, err)
		}
		out[i] = BoneFlexDriver{Bone: d.BoneIndex, Controls: controls}
	}
	return out, nil
}
