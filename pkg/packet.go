package multiplex

import (
	"encoding/binary"
	"io"
)

const (
	FSYNC  byte = 1
	FACK   byte = 2
	FFIN   byte = 4
	FRESET byte = 8
)

type packet struct {
	StreamID int16
	Flag     byte
	Data     []byte
}

func (p packet) Has(flags ...byte) bool {
	var aggregated byte = 0
	for _, f := range flags {
		aggregated |= f
	}
	return p.Flag&aggregated == aggregated
}

func encode(w io.Writer, p packet) error {
	errs := []error{
		binary.Write(w, binary.BigEndian, p.StreamID),
		binary.Write(w, binary.BigEndian, p.Flag),
		binary.Write(w, binary.BigEndian, int32(len(p.Data))),
	}
	for _, e := range errs {
		if e != nil {
			return e
		}
	}

	_, err := w.Write(p.Data)
	return err
}

func decode(r io.Reader, out *packet) error {
	var l int32
	errs := []error{
		binary.Read(r, binary.BigEndian, &out.StreamID),
		binary.Read(r, binary.BigEndian, &out.Flag),
		binary.Read(r, binary.BigEndian, &l),
	}
	for _, e := range errs {
		if e != nil {
			return e
		}
	}

	out.Data = make([]byte, l)
	return binary.Read(r, binary.BigEndian, &out.Data)
}
