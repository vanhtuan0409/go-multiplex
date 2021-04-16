package main

import (
	"errors"
	"net"
)

var (
	ErrHandleClosed = errors.New("handle closed")
)

type packet struct {
	handleId int
	flag     byte
	data     []byte
}

// Handle equal to a single connection
type Handle struct {
	id int

	in  <-chan packet
	out chan<- packet
}

func (h *Handle) Dial() error {
	return nil
}

func (h *Handle) Read(b []byte) (int, error) {
	p, ok := <-h.in
	if !ok {
		return 0, ErrHandleClosed
	}
	n := copy(b, p.data)
	return n, nil
}

func (h *Handle) Write(b []byte) (int, error) {
	select {

	case h.out <- packet{
		handleId: h.id,
		data:     b,
	}:
		return len(b), nil

	default:
		return 0, ErrHandleClosed

	}
}

func (h *Handle) Close() error {
	h.out <- packet{
		handleId: h.id,
		//todo add flag
	}
	return nil
}

func (h *Handle) send(p packet) error {
	return nil
}

func (h *Handle) receive() (packet, error) {
	return packet{}, nil
}

type MultiPlexClient struct {
	conn  net.Conn
	maxId int

	hs map[int]*Handle
}

func NewMultiplexClient(conn net.Conn) *MultiPlexClient {
	return &MultiPlexClient{
		conn: conn,
		hs:   make(map[int]*Handle),
	}
}

type MultiPlexServer struct {
	conn net.Conn

	hs map[int]*Handle
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	return &MultiPlexServer{
		conn: conn,
		hs:   make(map[int]*Handle),
	}
}
