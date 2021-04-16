package main

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrHandleClosed = errors.New("handle closed")
	ErrServerClosed = errors.New("server closed")
)

type packet struct {
	handleId int
	flag     byte
	data     []byte
}

// Handle equal to a single connection
type Handle struct {
	id int

	in  chan packet
	out chan packet

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

	h.send(packet{
		handleId: h.id,
		// flag here
	})

	close(h.in)
	close(h.out)

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

// NewHandle equal to Dial
func (c *MultiPlexClient) NewHandle() (*Handle, error) {
	h := &Handle{
		id:  c.maxId,
		in:  make(chan packet, 100),
		out: make(chan packet, 100),
	}
	c.maxId += 1
	if err := h.Dial(); err != nil {
		return nil, err
	}
	return h, nil
}

type MultiPlexServer struct {
	conn net.Conn

	backlog chan *Handle
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	return &MultiPlexServer{
		conn: conn,
	}
}

func (s *MultiPlexServer) Accept() (*Handle, error) {
	select {
	case h := <-s.backlog:
		return h, nil
	default:
		return nil, ErrServerClosed
	}
}
