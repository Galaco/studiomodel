package phy

import "io"

// ReadFromStream parses a phy from a io.Reader stream.
func ReadFromStream(stream io.Reader) (*Phy, error) {
	reader := NewReader()
	return reader.Read(stream)
}
