package main

import (
	"errors"
	"net"
	"sync"
)

var (
	ErrHandleClosed = errors.New("handle closed")
	ErrServerClosed = errors.New("server closed")

	FSYNC byte = 1
	FACK  byte = 10
	FFIN  byte = 100
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
	select {
	case h.out <- p:
		return nil
	default:
		return ErrHandleClosed
	}
}

func (h *Handle) receive() (packet, error) {
	select {
	case p := <-h.in:
		return p, nil
	default:
		return packet{}, ErrHandleClosed
	}
}

type MultiPlexClient struct {
	conn   net.Conn
	maxId  int
	bufOut chan packet

	hs map[int]*Handle
}

func NewMultiplexClient(conn net.Conn) *MultiPlexClient {
	c := &MultiPlexClient{
		conn:   conn,
		hs:     make(map[int]*Handle),
		bufOut: make(chan packet, 100),
	}
	go c.loopRead()
	go c.loopWrite()
	return c
}

// NewHandle equal to Dial
func (c *MultiPlexClient) NewHandle() (*Handle, error) {
	hOut := make(chan packet, 100)
	h := &Handle{
		id:  c.maxId,
		in:  make(chan packet, 100),
		out: hOut,
	}
	c.maxId += 1
	if err := h.Dial(); err != nil {
		return nil, err
	}

	// send to main loopWrite
	go func() {
		for p := range hOut {
			c.bufOut <- p
		}
	}()

	return h, nil
}

func (c *MultiPlexClient) loopRead() {
	for {
		p := &packet{}
		if err := decode(c.conn, p); err != nil {
			break
		}

		h, ok := c.hs[p.handleId]
		if !ok {
			// just ignore invalid packet
			continue
		}

		// handle multiple flag here

		h.in <- *p
	}
}

func (c *MultiPlexClient) loopWrite() {
	for p := range c.bufOut {
		// handle multiple packet flag here
		encode(c.conn, p)
	}
}

type MultiPlexServer struct {
	conn   net.Conn
	bufOut chan packet

	hs      map[int]*Handle
	backlog chan *Handle
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	s := &MultiPlexServer{
		conn:   conn,
		hs:     make(map[int]*Handle),
		bufOut: make(chan packet, 100),
	}
	go s.loopRead()
	go s.loopWrite()
	return s
}

func (s *MultiPlexServer) Accept() (*Handle, error) {
	select {
	case h := <-s.backlog:
		return h, nil
	default:
		return nil, ErrServerClosed
	}
}

func (s *MultiPlexServer) loopRead() {
	for {
		p := &packet{}
		if err := decode(s.conn, p); err != nil {
			break
		}

		h, ok := s.hs[p.handleId]
		if !ok {
			// just ignore invalid packet
			continue
		}

		// handle multiple flag here

		h.in <- *p
	}
}

func (s *MultiPlexServer) loopWrite() {
	for p := range s.bufOut {
		// handle multiple packet flag here
		encode(s.conn, p)
	}
}
