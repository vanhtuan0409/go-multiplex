package main

import (
	"errors"
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
	buf    chan []byte
	closed bool

	onWrite func(p packet) error
	onClose func()
}

func NewStream(id int) *Stream {
	return &Stream{
		id:  id,
		buf: make(chan []byte, 1000),
	}
}

func (s *Stream) Read(b []byte) (int, error) {
	if s.closed {
		return 0, ErrStreamClosed
	}

	received := <-s.buf
	n := copy(b, received)
	return n, nil
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
	close(s.buf)
}
