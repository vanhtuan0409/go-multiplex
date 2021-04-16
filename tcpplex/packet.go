package main

import (
	"encoding/gob"
	"io"
)

const (
	FSYNC byte = 1
	FACK  byte = 10
	FFIN  byte = 100
)

type packet struct {
	StreamID int
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
	encoder := gob.NewEncoder(w)
	return encoder.Encode(p)
}

func decode(r io.Reader, out *packet) error {
	decoder := gob.NewDecoder(r)
	return decoder.Decode(out)
}
