package main

import (
	"errors"
	"log"
	"net"
)

var (
	ErrServerClosed = errors.New("server closed")
)

type MultiPlexClient struct {
	conn    net.Conn
	m       *Conntrack
	backlog chan *Stream
}

func NewMultiplexClient(conn net.Conn) *MultiPlexClient {
	c := &MultiPlexClient{
		conn:    conn,
		m:       NewConntrackTable(),
		backlog: make(chan *Stream, 1000),
	}

	go c.loopRead()
	return c
}

func (c *MultiPlexClient) Dial() (*Stream, error) {
	// sending SYN for initial a new connection
	err := encode(c.conn, packet{
		StreamID: c.m.NextID(),
		Flag:     FSYNC,
		Data:     []byte{},
	})
	if err != nil {
		return nil, err
	}

	s := <-c.backlog
	return s, nil
}

func (c *MultiPlexClient) loopRead() {
	for {
		// parse received packet
		p := packet{}
		if err := decode(c.conn, &p); err != nil {
			log.Printf("[Client] Failed to parsed packet. ERR: %+v", err)
			break
		}

		// received SYN-ACK from remote, setup new stream
		// in fact, we shoud track multiple stage of connection handshake
		// not only as simple as this
		if p.Has(FSYNC, FACK) {
			s := c.createStream(p.StreamID)
			c.backlog <- s
			continue
		}

		// received packet not destine for this client
		// ignore (or send RESET)
		s := c.m.Lookup(p.StreamID)
		if s == nil {
			log.Printf("[Client] Dropped packet due to no recorded stream. Packet: +%v", p)
			continue
		}

		// received FIN from remote, passively close stream
		if p.Has(FFIN) {
			s.destroy()
			continue
		}

		// exchange data between 2 end
		s.pWrite.Write(p.Data)
	}
}

func (c *MultiPlexClient) createStream(id int) *Stream {
	s := NewStream(id)
	s.onWrite = func(p packet) error {
		return encode(c.conn, p)
	}
	c.m.Register(s)
	return s
}

type MultiPlexServer struct {
	conn net.Conn
	m    *Conntrack

	backlog chan *Stream
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	sv := &MultiPlexServer{
		conn:    conn,
		m:       NewConntrackTable(),
		backlog: make(chan *Stream, 1000),
	}

	go sv.loopRead()
	return sv
}

func (sv *MultiPlexServer) Accept() *Stream {
	return <-sv.backlog
}

func (sv *MultiPlexServer) loopRead() {
	for {
		// parse received packet
		p := packet{}
		if err := decode(sv.conn, &p); err != nil {
			log.Printf("[Server] Failed to parsed packet. ERR: %+v", err)
			break
		}

		// receive SYN packet, return SYN-ACK
		if p.Has(FSYNC) {
			encode(sv.conn, packet{
				StreamID: p.StreamID,
				Flag:     FSYNC | FACK,
				Data:     []byte{},
			})
			s := sv.createStream(p.StreamID)
			sv.backlog <- s
			continue
		}

		// received packet not destine for this server
		// ignore (or send RESET)
		s := sv.m.Lookup(p.StreamID)
		if s == nil {
			log.Printf("[Server] Dropped packet due to no recorded stream. Packet: +%v", p)
			continue
		}

		// received FIN from remote, passively close stream
		if p.Has(FFIN) {
			s.destroy()
			continue
		}

		// exchange data between 2 end
		s.pWrite.Write(p.Data)

	}
}

func (sv *MultiPlexServer) createStream(id int) *Stream {
	s := NewStream(id)
	s.onWrite = func(p packet) error {
		return encode(sv.conn, p)
	}
	sv.m.Register(s)
	return s
}
