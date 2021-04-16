package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

var (
	port = 7575
)

func worker(id int, conn *Stream, stat chan<- bool) {
	clientId := fmt.Sprintf("client%d", id)
	r := bufio.NewReader(conn)
	for {
		msg := append([]byte(clientId), '\n')
		conn.Write(msg)
		resp, _ := r.ReadString('\n')
		resp = strings.TrimSpace(resp)
		fmt.Printf("[%s] Received `%s`\n", clientId, resp)
		stat <- (resp == clientId) // send stats
		time.Sleep(time.Second)
	}
}

func client() {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	stats := make(chan bool, 100)

	plexClient := NewMultiplexClient(conn)
	numClient := 5
	var wg sync.WaitGroup
	wg.Add(numClient)
	for i := 0; i < numClient; i++ {
		go func(id int) {
			defer wg.Done()
			stream, err := plexClient.Dial()
			if err != nil {
				log.Printf("Failed to create stream. ERR: %v", err)
				return
			}

			worker(id, stream, stats)
		}(i)
	}

	wg.Wait()
}

func server() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	log.Printf("Start listening on :%d", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		plexServer := NewMultiplexServer(conn)
		for {
			stream := plexServer.Accept()
			go func(s *Stream) {
				defer s.Close()
				r := bufio.NewReader(s)
				for {
					line, err := r.ReadBytes('\n')
					if err != nil {
						break
					}
					if _, err := s.Write(line); err != nil {
						break
					}
				}

			}(stream)
		}
	}
}

func main() {
	go server()
	time.Sleep(time.Second)
	client()
}
