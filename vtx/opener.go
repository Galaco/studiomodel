package vtx

import "io"

func ReadFromStream(stream io.Reader) (*Vtx, error) {
	reader := NewReader()
	return reader.Read(stream)
}
