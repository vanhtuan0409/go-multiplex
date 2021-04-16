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

	outBuf chan packet
}

func NewMultiplexClient(conn net.Conn) *MultiPlexClient {
	c := &MultiPlexClient{
		conn:    conn,
		m:       NewConntrackTable(),
		backlog: make(chan *Stream, 1000),
		outBuf:  make(chan packet, 1000),
	}

	go c.loopWrite()
	go c.loopRead()
	return c
}

func (c *MultiPlexClient) Dial() (*Stream, error) {
	// sending SYN for initial a new connection
	c.outBuf <- packet{
		StreamID: c.m.NextID(),
		Flag:     FSYNC,
		Data:     []byte{},
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

func (c *MultiPlexClient) loopWrite() {
	for p := range c.outBuf {
		if err := encode(c.conn, p); err != nil {
			log.Printf("[Client] Failed to send packet. ERR: %+v", err)
		}
	}
}

func (c *MultiPlexClient) createStream(id int16) *Stream {
	s := NewStream(id)
	s.out = c.outBuf
	c.m.Register(s)
	return s
}

type MultiPlexServer struct {
	conn net.Conn
	m    *Conntrack

	backlog chan *Stream
	outBuf  chan packet
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	sv := &MultiPlexServer{
		conn:    conn,
		m:       NewConntrackTable(),
		backlog: make(chan *Stream, 1000),
		outBuf:  make(chan packet, 1000),
	}

	go sv.loopWrite()
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
			sv.outBuf <- packet{
				StreamID: p.StreamID,
				Flag:     FSYNC | FACK,
				Data:     []byte{},
			}
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

func (sv *MultiPlexServer) loopWrite() {
	for p := range sv.outBuf {
		if err := encode(sv.conn, p); err != nil {
			log.Printf("[Server] Failed to send packet. ERR: %+v", err)
		}
	}
}

func (sv *MultiPlexServer) createStream(id int16) *Stream {
	s := NewStream(id)
	s.out = sv.outBuf
	sv.m.Register(s)
	return s
}
