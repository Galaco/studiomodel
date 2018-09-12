package mdl

import "io"

func ReadFromStream(stream io.Reader) (*Mdl, error){
	reader := Reader{
		stream: stream,
	}
	return reader.Read()
}
