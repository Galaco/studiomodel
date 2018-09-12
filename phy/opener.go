package phy

import "io"

func ReadFromStream(stream io.Reader) (*Phy, error) {
	reader := Reader{
		stream: stream,
	}
	return reader.Read()
}
