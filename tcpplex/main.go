package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	multiplex "github.com/vanhtuan0409/go-multiplex"
)

var (
	port = 7575

	stats *multiplex.Stats
)

type ClientFactory struct {
	client *MultiPlexClient
}

func NewClientFactory() *ClientFactory {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	return &ClientFactory{
		client: NewMultiplexClient(conn),
	}
}

func (f *ClientFactory) GetNewClient(id int) (*multiplex.Worker, error) {
	stream, err := f.client.Dial()
	if err != nil {
		return nil, err
	}

	return &multiplex.Worker{
		ID:     id,
		Conn:   stream,
		Atomic: false,
	}, nil
}

type PlexListener struct {
	c  chan multiplex.Conn
	nl net.Listener
}

func NewPlexListener() *PlexListener {
	nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	log.Printf("Server running at: %d", port)

	return &PlexListener{
		nl: nl,
		c:  make(chan multiplex.Conn, 1024),
	}

}

func (l *PlexListener) Run() {
	for {
		conn, err := l.nl.Accept()
		if err != nil {
			break
		}

		plexServer := NewMultiplexServer(conn)
		for {
			s := plexServer.Accept()
			l.c <- s
		}
	}
}

func (l *PlexListener) Accept() (multiplex.Conn, error) {
	conn := <-l.c
	return conn, nil
}

func main() {
	server := multiplex.Server{
		L: NewPlexListener(),
	}
	go server.Run()

	time.Sleep(time.Second)
	generator := multiplex.LoadGenerator{
		NumWorker: 5,
		C:         NewClientFactory(),
		Stats:     multiplex.NewStats(time.Second),
	}
	generator.Run(context.Background())
}
