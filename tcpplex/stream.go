package main

import (
	"errors"
	"io"
	"time"
)

var (
	ErrStreamClosed = errors.New("stream closed")
	ErrTimeout      = errors.New("timeout")

	defaultTimeout = 5 * time.Second
)

// Stream equal to a single connection
type Stream struct {
	id     int
	closed bool

	pRead  *io.PipeReader
	pWrite *io.PipeWriter

	onWrite func(p packet) error
	onClose func()
}

func NewStream(id int) *Stream {
	pr, pw := io.Pipe()
	return &Stream{
		id:     id,
		pRead:  pr,
		pWrite: pw,
	}
}

func (s *Stream) Read(b []byte) (int, error) {
	if s.closed {
		return 0, ErrStreamClosed
	}
	return s.pRead.Read(b)
}

func (s *Stream) Write(b []byte) (int, error) {
	if s.closed {
		return 0, ErrStreamClosed
	}
	p := packet{
		StreamID: s.id,
		Data:     b,
	}
	if err := s.onWrite(p); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close active close connection
func (s *Stream) Close() error {
	if s.closed {
		return ErrStreamClosed
	}
	s.onWrite(packet{
		StreamID: s.id,
		Flag:     FFIN,
		Data:     []byte{},
	})
	s.destroy()
	return nil
}

func (s *Stream) destroy() {
	s.closed = true
	s.onClose()
	s.pWrite.Close()
	s.pWrite.Close()
}
