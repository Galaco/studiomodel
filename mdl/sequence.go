package mdl

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/galaco/studiomodel/internal"
)

// SequenceData is the detail of one sequence
type SequenceData struct {
	// Events
	Events []SequenceEvent
	// Animations is the blend grid: GroupSize[0] columns by GroupSize[1] rows of indices into AnimDescs, row by row.
	// Animations from included models count after this model's own.
	Animations []int16
	// PoseKeys are the pose parameter values at each column, then each row, of the blend grid
	PoseKeys []float32
	// AutoLayers are other sequences played on top of this one
	AutoLayers []AutoLayer
	// BoneWeights are how much the sequence moves each bone
	BoneWeights []float32
	// IKLocks
	IKLocks []IKLock
	// KeyValues
	KeyValues string
	// ActivityModifiers are names that select between sequences of the same activity
	ActivityModifiers []string
}

// eventNewStyle marks an Event whose Name identifies it, rather than its number
const eventNewStyle = 1 << 10

// event is mstudioevent_t in studio.h
type event struct {
	Cycle     float32
	Event     int32
	Type      int32
	Options   [64]byte
	NameIndex int32
}

// SequenceEvent is something a sequence triggers at a point in its cycle, such as a footstep sound
type SequenceEvent struct {
	// Cycle - when the event fires, from 0 to 1
	Cycle float32
	// Event - the event's number, for older events without a Name
	Event int32
	// Type
	Type int32
	// Options - the event's parameter, such as a sound name
	Options string
	// Name - the event's name
	Name string
}

// AutoLayer plays another sequence on top of a sequence
// Corresponds to mstudioautolayer_t in studio.h
type AutoLayer struct {
	// Sequence
	Sequence int16
	// Pose - the pose parameter driving the layer's cycle, for layers that use one
	Pose int16
	// Flags
	Flags int32
	// Start, Peak, Tail and End are cycles: the layer fades in from Start to Peak and out from Tail to End
	Start float32
	Peak  float32
	Tail  float32
	End   float32
}

// readSequences reads the detail of each sequence. Offsets are relative to each sequence's description.
func readSequences(buf []byte, header *Studiohdr, out *Mdl) error {
	descSize := int64(binary.Size(SequenceDesc{}))
	eventSize := int64(binary.Size(event{}))
	out.Sequences = make([]SequenceData, len(out.SequenceDescs))
	for i, desc := range out.SequenceDescs {
		start := int64(header.LocalSequenceOffset) + int64(i)*descSize
		seq := &out.Sequences[i]
		fail := func(err error) error {
			return fmt.Errorf("sequence %d: %w", i, err)
		}

		events, err := internal.ReadSlice[event](buf, start+int64(desc.EventIndex), int64(desc.NumEvents), "events")
		if err != nil {
			return fail(err)
		}
		seq.Events = make([]SequenceEvent, len(events))
		for j, e := range events {
			seq.Events[j] = SequenceEvent{Cycle: e.Cycle, Event: e.Event, Type: e.Type, Options: fixedString(e.Options[:])}
			if e.NameIndex != 0 {
				if seq.Events[j].Name, err = internal.CString(buf, start+int64(desc.EventIndex)+int64(j)*eventSize+int64(e.NameIndex), 256, "event name"); err != nil {
					return fail(err)
				}
			}
		}

		if desc.GroupSize[0] < 0 || desc.GroupSize[1] < 0 {
			return fail(fmt.Errorf("invalid blend grid size %d by %d", desc.GroupSize[0], desc.GroupSize[1]))
		}
		if seq.Animations, err = internal.ReadSlice[int16](buf, start+int64(desc.AnimIndexIndex), int64(desc.GroupSize[0])*int64(desc.GroupSize[1]), "blend grid"); err != nil {
			return fail(err)
		}
		if desc.PoseKeyIndex != 0 {
			if seq.PoseKeys, err = internal.ReadSlice[float32](buf, start+int64(desc.PoseKeyIndex), int64(desc.GroupSize[0])+int64(desc.GroupSize[1]), "pose keys"); err != nil {
				return fail(err)
			}
		}
		if seq.AutoLayers, err = internal.ReadSlice[AutoLayer](buf, start+int64(desc.AutoLayerIndex), int64(desc.NumAutoLayers), "auto layers"); err != nil {
			return fail(err)
		}
		if desc.WeightListIndex != 0 {
			if seq.BoneWeights, err = internal.ReadSlice[float32](buf, start+int64(desc.WeightListIndex), int64(len(out.Bones)), "bone weights"); err != nil {
				return fail(err)
			}
		}
		if seq.IKLocks, err = internal.ReadSlice[IKLock](buf, start+int64(desc.IKLockIndex), int64(desc.NumIKLocks), "IK locks"); err != nil {
			return fail(err)
		}
		if desc.KeyValueSize > 0 {
			kvStart, kvSize := start+int64(desc.KeyValueIndex), int64(desc.KeyValueSize)
			if err := internal.CheckRange(buf, kvStart, kvSize, "key values"); err != nil {
				return fail(err)
			}
			seq.KeyValues = strings.TrimRight(string(buf[kvStart:kvStart+kvSize]), "\x00")
		}

		// Each activity modifier is a name offset, relative to itself
		modifiersStart := start + int64(desc.ActivityModifierIndex)
		modifiers, err := internal.ReadSlice[int32](buf, modifiersStart, int64(desc.NumActivityModifiers), "activity modifiers")
		if err != nil {
			return fail(err)
		}
		if seq.ActivityModifiers, err = readNames(buf, modifiersStart, modifiers, func(nameIndex int32) int32 { return nameIndex }, "activity modifier"); err != nil {
			return fail(err)
		}
	}
	return nil
}

// readTransitions reads the animation transition graph: its node names, and the table of transitions between nodes
func readTransitions(buf []byte, header *Studiohdr, out *Mdl) error {
	numNodes := int64(header.LocalNodeCount)
	if numNodes <= 0 {
		return nil
	}

	// Name offsets are relative to the header
	nameOffsets, err := internal.ReadSlice[int32](buf, int64(header.LocalNodeNameIndex), numNodes, "node names")
	if err != nil {
		return err
	}
	out.NodeNames = make([]string, len(nameOffsets))
	for i, offset := range nameOffsets {
		if out.NodeNames[i], err = internal.CString(buf, int64(offset), 256, "node name"); err != nil {
			return err
		}
	}

	table, err := internal.ReadSlice[uint8](buf, int64(header.LocalNodeIndex), numNodes*numNodes, "transitions")
	if err != nil {
		return err
	}
	out.Transitions = make([][]uint8, numNodes)
	for i := range out.Transitions {
		out.Transitions[i] = table[int64(i)*numNodes : int64(i+1)*numNodes]
	}
	return nil
}
