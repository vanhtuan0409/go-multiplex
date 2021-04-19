package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	multiplex "github.com/vanhtuan0409/go-multiplex"
)

var (
	port = 7575
	lock bool
)

type netConn struct {
	net.Conn
	sync.Mutex
}

type ClientFactory struct {
	conn *netConn
}

func NewClientFactory() *ClientFactory {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	return &ClientFactory{
		conn: &netConn{Conn: conn},
	}
}

func (f *ClientFactory) GetNewClient(id int) (*multiplex.Worker, error) {
	return &multiplex.Worker{
		ID:     id,
		Conn:   f.conn,
		Atomic: lock,
	}, nil
}

type NetListener struct {
	c chan multiplex.Conn
}

func (l *NetListener) Run() {
	nl, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	log.Printf("Server running at: %d", port)

	for {
		conn, err := nl.Accept()
		if err != nil {
			break
		}

		l.c <- &netConn{Conn: conn}
	}
}

func (l *NetListener) Accept() (multiplex.Conn, error) {
	conn := <-l.c
	return conn, nil
}

func main() {
	flag.BoolVar(&lock, "lock", false, "Perform lock on request")
	flag.Parse()

	server := multiplex.Server{
		L: &NetListener{
			c: make(chan multiplex.Conn, 1024),
		},
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
