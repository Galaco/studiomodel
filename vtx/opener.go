package vtx

import "io"

// ReadFromStream parses a vtx from an io.Reader stream.
// The vtx doesn't say whether its strip headers use the larger MDL v49 layout, so it is detected from the data, and
// Vtx.StripHeaders reports the layout and how it was chosen. When the mdl is at hand, ReadFromStreamWithMDLVersion
// chooses by its version instead.
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
