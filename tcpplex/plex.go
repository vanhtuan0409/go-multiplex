package main

import (
	"errors"
	"net"
)

var (
	ErrServerClosed = errors.New("server closed")
)

type MultiPlexClient struct {
	t *Transport
}

func NewMultiplexClient(conn net.Conn) *MultiPlexClient {
	return &MultiPlexClient{
		t: NewTransport(conn, TransportOptions{
			LogPrefix: "Client",
		}),
	}
}

func (c *MultiPlexClient) Dial() (*Stream, error) {
	c.t.outBuf <- packet{
		StreamID: c.t.m.NextID(),
		Flag:     FSYNC,
		Data:     []byte{},
	}
	s := <-c.t.outBacklog
	return s, nil
}

type MultiPlexServer struct {
	t *Transport
}

func NewMultiplexServer(conn net.Conn) *MultiPlexServer {
	return &MultiPlexServer{
		t: NewTransport(conn, TransportOptions{
			LogPrefix:     "Server",
			EnableBinding: true,
		}),
	}
}

func (sv *MultiPlexServer) Accept() *Stream {
	return <-sv.t.inBacklog
}
