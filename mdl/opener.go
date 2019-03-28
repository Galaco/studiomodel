package mdl

import "io"

// ReadFromStream parses an mdl from an io.Reader stream
func ReadFromStream(stream io.Reader) (*Mdl, error) {
	reader := NewReader()
	return reader.Read(stream)
}
