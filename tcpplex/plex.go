package main

import (
	"errors"
	"net"
	"sync"
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

	sync.Mutex
}

func (h *Handle) Dial() error {
	h.Lock()
	defer h.Lock()

	return nil
}

func (h *Handle) Read(b []byte) (int, error) {
	h.Lock()
	defer h.Lock()

	p, err := h.receive()
	if err != nil {
		return 0, err
	}
	n := copy(b, p.data)
	return n, nil
}

func (h *Handle) Write(b []byte) (int, error) {
	h.Lock()
	defer h.Lock()

	err := h.send(packet{
		handleId: h.id,
		data:     b,
	})
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (h *Handle) Close() error {
	h.Lock()
	defer h.Lock()

	return nil
}

func (h *Handle) send(p packet) error {
	// can implement retry and SYN-ACK here
	return nil
}

func (h *Handle) receive() (packet, error) {
	// can implement retry and SYN-ACK here
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
