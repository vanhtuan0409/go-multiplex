package main

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrHandleClosed = errors.New("handle closed")
	ErrTimeout      = errors.New("timeout")
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
	h.send(packet{
		handleId: h.id,
		flag:     FSYNC,
		data:     []byte{},
	})

	p, err := h.receive()
	if err != nil {
		return err
	}

	// Chec SYN-ACK
	fsynack := FSYNC & FACK
	if p.flag&fsynack != fsynack {
		return ErrHandleClosed
	}

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

// Close active close connection
func (h *Handle) Close() error {
	h.Lock()
	defer h.Lock()

	h.send(packet{
		handleId: h.id,
		flag:     FFIN,
		data:     []byte{},
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
	ticker := time.NewTimer(5 * time.Second)
	select {
	case p := <-h.in:
		return p, nil
	case <-ticker.C:
		return packet{}, ErrTimeout
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
	h := &Handle{
		id:  c.maxId,
		in:  make(chan packet, 100),
		out: make(chan packet, 100),
	}

	c.maxId += 1 // should generate uuid
	if err := h.Dial(); err != nil {
		return nil, err
	}

	// send to main loopWrite
	go func() {
		for p := range h.out {
			c.bufOut <- p
		}
	}()

	c.hs[h.id] = h
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

		// handle passive close
		if p.flag&FFIN == FFIN {
			close(h.in)
			close(h.out)
			delete(c.hs, p.handleId)
			return
		}

		h.in <- *p
	}
}

func (c *MultiPlexClient) loopWrite() {
	for p := range c.bufOut {
		// handle active close
		if p.flag&FFIN == FFIN {
			delete(c.hs, p.handleId)
		}

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
			continue
		}

		// handle SYNC new connection
		if p.flag&FSYNC == FSYNC {
			// send back SYN-ACK
			h := Handle{
				id:  p.handleId,
				in:  make(chan packet, 100),
				out: make(chan packet, 100),
			}
			go func() {
				for p := range h.out {
					s.bufOut <- p
				}
			}()
			h.send(packet{
				handleId: p.handleId,
				flag:     FSYNC & FACK,
				data:     []byte{},
			})
			s.hs[h.id] = &h
			s.backlog <- &h
			return
		}

		// handle passive close
		if p.flag&FFIN == FFIN {
			close(h.in)
			close(h.out)
			delete(s.hs, p.handleId)
			return
		}

		h.in <- *p
	}
}

func (s *MultiPlexServer) loopWrite() {
	for p := range s.bufOut {
		// handle active close
		if p.flag&FFIN == FFIN {
			delete(s.hs, p.handleId)
		}

		encode(s.conn, p)
	}
}
