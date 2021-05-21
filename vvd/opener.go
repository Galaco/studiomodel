package vvd

import "io"

// ReadFromStream parses a vvd from a io.Reader stream.
func ReadFromStream(stream io.Reader) (*Vvd, error) {
	reader := NewReader()
	return reader.Read(stream)
}
