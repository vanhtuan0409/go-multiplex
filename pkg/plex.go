package multiplex

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

func NewMultiplexClient(network, address string) (*MultiPlexClient, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return &MultiPlexClient{
		t: NewTransport(conn, TransportOptions{
			LogPrefix: "Client",
		}),
	}, nil
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
	l net.Listener
	c chan *Stream
}

func NewMultiplexServer(network, address string) (*MultiPlexServer, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	sv := &MultiPlexServer{
		l: l,
		c: make(chan *Stream, 1024),
	}
	go sv.run()

	return sv, nil
}

func (sv *MultiPlexServer) Accept() (*Stream, error) {
	return <-sv.c, nil
}

func (sv *MultiPlexServer) run() {
	for {
		conn, err := sv.l.Accept()
		if err != nil {
			break
		}

		tp := NewTransport(conn, TransportOptions{
			LogPrefix:     "Server",
			EnableBinding: true,
		})
		go copyChan(tp.inBacklog, sv.c)
	}
}
