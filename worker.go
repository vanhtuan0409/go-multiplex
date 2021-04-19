package multiplex

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
)

type Conn interface {
	io.ReadWriteCloser
	sync.Locker
}

type Worker struct {
	ID     int
	Conn   Conn
	Atomic bool
}

func (w *Worker) Run(ctx context.Context, stats *Stats) {
	defer w.Conn.Close()
	clientId := fmt.Sprintf("client%d", w.ID)
	r := bufio.NewReader(w.Conn)
	for {
		select {
		case <-ctx.Done():
			break

		default:
			func() {
				if w.Atomic {
					w.Conn.Lock()
					defer w.Conn.Unlock()
				}

				msg := append([]byte(clientId), '\n')
				w.Conn.Write(msg)
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
}

type Client interface {
	GetNewClient(id int) (*Worker, error)
}

type LoadGenerator struct {
	NumWorker int
	C         Client
	Stats     *Stats
}

func (l *LoadGenerator) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(l.NumWorker)

	for i := 0; i < l.NumWorker; i++ {
		worker, err := l.C.GetNewClient(i)
		if err != nil {
			log.Printf("Failed to create new client. ERR: %+v", err)
			continue
		}

		go func() {
			defer wg.Done()
			worker.Run(ctx, l.Stats)
		}()
	}

	wg.Wait()
}
