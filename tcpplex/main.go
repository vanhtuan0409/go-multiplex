package main

import (
	"context"
	"fmt"
	"time"

	util "github.com/vanhtuan0409/go-multiplex"
	multiplex "github.com/vanhtuan0409/go-multiplex/pkg"
)

var (
	port = 7575

	stats *util.Stats
)

type ClientFactory struct {
	client *multiplex.MultiPlexClient
}

func NewClientFactory() *ClientFactory {
	client, err := multiplex.NewMultiplexClient("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	return &ClientFactory{
		client: client,
	}
}

func (f *ClientFactory) GetNewClient(id int) (*util.Worker, error) {
	stream, err := f.client.Dial()
	if err != nil {
		return nil, err
	}

	return &util.Worker{
		ID:     id,
		Conn:   stream,
		Atomic: false,
	}, nil
}

func server() {
	l, err := multiplex.NewMultiplexServer("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			break
		}
		go util.HandleConn(conn)
	}
}

func main() {
	go server()

	time.Sleep(time.Second)
	generator := util.LoadGenerator{
		NumWorker: 5,
		C:         NewClientFactory(),
		Stats:     util.NewStats(time.Second),
	}
	generator.Run(context.Background())
}
