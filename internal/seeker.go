package internal

import (
	"reflect"
)

// Seeker
type Seeker struct {
	buf      *[]byte
	Position int32
	Begin    int32
}

// Seek
func (s *Seeker) Seek(offset int32, start int32) {
	s.Position = start + offset
}

// Read
func (s *Seeker) Read(num int32, size int32, callback func([]byte)) {
	s.Position += num * size
	callback((*s.buf)[s.Position-(num*size) : s.Position])
}

// NewSeeker
func NewSeeker(buf *[]byte) *Seeker {
	return &Seeker{
		buf:      buf,
		Position: 0,
		Begin:    0,
	}
}

// SizeOf
func SizeOf(t interface{}) int32 {
	typeName := reflect.TypeOf(t)
	return int32(typeName.Elem().Size())
}
