package vtx

import "io"

// ReadFromStream parses a vtx from an io.Reader stream.
// Whether strip headers use the larger MDL v49 layout is detected from the data.
func ReadFromStream(stream io.Reader) (*Vtx, error) {
	reader := NewReader()
	return reader.Read(stream)
}

// ReadFromStreamWithMDLVersion parses a vtx from an io.Reader stream, using the strip header layout of the given mdl
// version instead of detecting it.
func ReadFromStreamWithMDLVersion(stream io.Reader, mdlVersion int32) (*Vtx, error) {
	reader := NewReader()
	reader.MDLVersion = mdlVersion
	return reader.Read(stream)
}
