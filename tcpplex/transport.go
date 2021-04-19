package main

import (
	"log"
	"net"
)

type TransportOptions struct {
	LogPrefix     string
	EnableBinding bool
}

type Transport struct {
	options TransportOptions

	conn net.Conn   // underlying data-link
	m    *Conntrack // streams book-keeping

	outBuf     chan packet  // out-going packet buffer, waiting to be sent to the wire
	outBacklog chan *Stream // backlog for initialized Streams
	inBacklog  chan *Stream // backlog for Streams waiting to be accepted
}

func NewTransport(conn net.Conn, options TransportOptions) *Transport {
	if options.LogPrefix == "" {
		options.LogPrefix = "Transport"
	}

	t := &Transport{
		options: options,

		conn: conn,
		m:    NewConntrackTable(),

		outBuf:     make(chan packet, 1024),
		outBacklog: make(chan *Stream, 1024),
		inBacklog:  make(chan *Stream, 1024),
	}
	go t.loopWrite()
	go t.loopRead()

	return t
}

func (t *Transport) loopRead() {
	for {
		// parse received packet
		p := packet{}
		if err := decode(t.conn, &p); err != nil {
			log.Printf("[%s] Failed to parsed packet. ERR: %+v", t.options.LogPrefix, err)
			break
		}

		// received SYN-ACK from remote, setup new stream
		// in fact, we shoud track multiple stage of connection handshake
		// not only as simple as this
		if p.Has(FSYNC, FACK) {
			s := t.createStream(p.StreamID)
			t.outBacklog <- s
			continue
		}

		// receive SYN packet, return SYN-ACK
		// if not enable to accept new Stream, send back reset
		if p.Has(FSYNC) && t.options.EnableBinding {
			t.outBuf <- packet{
				StreamID: p.StreamID,
				Flag:     FSYNC | FACK,
				Data:     []byte{},
			}
			s := t.createStream(p.StreamID)
			t.inBacklog <- s
			continue
		} else if p.Has(FSYNC) {
			t.outBuf <- packet{
				StreamID: p.StreamID,
				Flag:     FRESET,
				Data:     []byte{},
			}
			continue
		}

		// received packet not destine for this client
		// send back RESET
		s := t.m.Lookup(p.StreamID)
		if s == nil {
			log.Printf("[%s] Dropped packet due to no recorded stream. Packet: +%v", t.options.LogPrefix, p)
			t.outBuf <- packet{
				StreamID: p.StreamID,
				Flag:     FRESET,
				Data:     []byte{},
			}
			continue
		}

		// received FIN or RESET from remote, passively close stream
		if p.Has(FFIN) || p.Has(FRESET) {
			s.destroy()
			continue
		}

		// copy data from network buffer into
		// stream memory
		s.pWrite.Write(p.Data)
	}
}

func (t *Transport) loopWrite() {
	for p := range t.outBuf {
		if err := encode(t.conn, p); err != nil {
			log.Printf("[%s] Failed to send packet. ERR: %+v", t.options.LogPrefix, err)
		}
	}
}

func (t *Transport) createStream(id int16) *Stream {
	s := NewStream(id)
	s.out = t.outBuf
	t.m.Register(s)
	return s
}
