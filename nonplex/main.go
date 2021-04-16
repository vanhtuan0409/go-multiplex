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
)

var (
	port = 7575
	lock bool
)

type netConn struct {
	net.Conn
	sync.Mutex
}

func worker(id int, conn *netConn, stat chan<- bool) {
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
			stat <- (resp == clientId) // send stats
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

	stats := make(chan bool, 100)
	ticker := time.NewTicker(time.Second)
	matched := 0
	unmatched := 0
	go func() {
		for {
			select {
			case isMatched := <-stats:
				if isMatched {
					matched += 1
				} else {
					unmatched += 1
				}
			case <-ticker.C:
				log.Printf("Stats per sec. Matched: %d, Unmatched: %d, Total: %d", matched, unmatched, matched+unmatched)
				matched = 0
				unmatched = 0
			}
		}
	}()

	for i := 0; i < numClient; i++ {
		go func(id int) {
			defer wg.Done()
			worker(id, nc, stats)
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
				line, err := r.ReadString('\n')
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

	go server()
	time.Sleep(time.Second)
	client()
}
