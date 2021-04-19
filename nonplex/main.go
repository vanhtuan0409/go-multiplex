package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/vanhtuan0409/go-multiplex"
	util "github.com/vanhtuan0409/go-multiplex"
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

func (f *ClientFactory) GetNewClient(id int) (*util.Worker, error) {
	return &util.Worker{
		ID:     id,
		Conn:   f.conn,
		Atomic: lock,
	}, nil
}

func server() {
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

		go util.HandleConn(conn)
	}
}

func main() {
	flag.BoolVar(&lock, "lock", false, "Perform lock on request")
	flag.Parse()

	go server()

	time.Sleep(time.Second)
	generator := multiplex.LoadGenerator{
		NumWorker: 5,
		C:         NewClientFactory(),
		Stats:     multiplex.NewStats(time.Second),
	}
	generator.Run(context.Background())
}
