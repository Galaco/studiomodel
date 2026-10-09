// Package testutil builds little-endian test files byte by byte, so tests do not depend on the struct
// definitions they are checking.
package testutil

import (
	"encoding/binary"
	"math"
)

// Builder is an append-only little-endian byte buffer
type Builder struct {
	Bytes []byte
}

// Pos returns the offset the next write will land at
func (b *Builder) Pos() int {
	return len(b.Bytes)
}

// I32 appends int32s
func (b *Builder) I32(values ...int32) *Builder {
	for _, v := range values {
		b.Bytes = binary.LittleEndian.AppendUint32(b.Bytes, uint32(v))
	}
	return b
}

// I16 appends int16s
func (b *Builder) I16(values ...int16) *Builder {
	for _, v := range values {
		b.Bytes = binary.LittleEndian.AppendUint16(b.Bytes, uint16(v))
	}
	return b
}

// F32 appends float32s
func (b *Builder) F32(values ...float32) *Builder {
	for _, v := range values {
		b.Bytes = binary.LittleEndian.AppendUint32(b.Bytes, math.Float32bits(v))
	}
	return b
}

// U8 appends bytes
func (b *Builder) U8(values ...byte) *Builder {
	b.Bytes = append(b.Bytes, values...)
	return b
}

// Str appends a null-terminated string
func (b *Builder) Str(s string) *Builder {
	b.Bytes = append(append(b.Bytes, s...), 0)
	return b
}

// Zero appends n zero bytes
func (b *Builder) Zero(n int) *Builder {
	b.Bytes = append(b.Bytes, make([]byte, n)...)
	return b
}

// PadTo appends zero bytes until the length is a multiple of align
func (b *Builder) PadTo(align int) *Builder {
	for len(b.Bytes)%align != 0 {
		b.Bytes = append(b.Bytes, 0)
	}
	return b
}

// SetI32 overwrites the int32 at offset
func (b *Builder) SetI32(offset int, v int32) {
	binary.LittleEndian.PutUint32(b.Bytes[offset:], uint32(v))
}

// SetF32 overwrites the float32 at offset
func (b *Builder) SetF32(offset int, v float32) {
	binary.LittleEndian.PutUint32(b.Bytes[offset:], math.Float32bits(v))
}
