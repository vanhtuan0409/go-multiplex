package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	multiplex "github.com/vanhtuan0409/go-multiplex"
)

var (
	port = 7575
	lock bool

	stats *multiplex.Stats
)

type netConn struct {
	net.Conn
	sync.Mutex
}

func worker(id int, conn *netConn) {
	clientId := fmt.Sprintf("client%d", id)
	r := bufio.NewReader(conn)
	for {
		func() {
			if lock {
				conn.Lock()
				defer conn.Unlock()
			}

			conn.Write(append([]byte(clientId), '\n'))
			resp, _ := r.ReadString('\n')
			resp = strings.TrimSpace(resp)

			stats.Record("total", 1)
			if clientId == resp {
				stats.Record("matched", 1)
			} else {
				stats.Record("unmatched", 1)
			}
		}()
	}
}

func client() {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	nc := &netConn{
		Conn: conn,
	}

	numClient := 5
	var wg sync.WaitGroup
	wg.Add(numClient)
	for i := 0; i < numClient; i++ {
		go func(id int) {
			defer wg.Done()
			worker(id, nc)
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

		// Connection handler
		go func(c net.Conn) {
			defer c.Close()
			r := bufio.NewReader(c)
			for {
				line, err := r.ReadBytes('\n')
				if err != nil {
					break
				}
				if _, err := c.Write([]byte(line)); err != nil {
					break
				}
			}
		}(conn)
	}
}

func main() {
	flag.BoolVar(&lock, "lock", false, "Perform lock on request")
	flag.Parse()

	stats = multiplex.NewStats(time.Second)
	go server()
	time.Sleep(time.Second)
	client()
}
