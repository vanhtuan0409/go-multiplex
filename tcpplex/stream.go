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
	id     int16
	closed bool

	pRead  *io.PipeReader // use as input source for Stream's read
	pWrite *io.PipeWriter // use as a output destination for Transport when receiving a packet
	out    chan packet    // use as output destination for Stream's write

	onClose func()
}

func NewStream(id int16) *Stream {
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
	s.out <- packet{
		StreamID: s.id,
		Data:     b,
	}
	return len(b), nil
}

// Close active close connection
func (s *Stream) Close() error {
	if s.closed {
		return ErrStreamClosed
	}
	s.out <- packet{
		StreamID: s.id,
		Flag:     FFIN,
		Data:     []byte{},
	}
	s.destroy()
	return nil
}

func (s *Stream) destroy() {
	s.closed = true
	s.onClose()
	s.pWrite.Close()
	s.pWrite.Close()
}
