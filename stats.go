package multiplex

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type record struct {
	name  string
	delta int
}

type Stats struct {
	c      chan record
	stats  map[string]int
	ticker *time.Ticker
}

func NewStats(interval time.Duration) *Stats {
	s := &Stats{
		c:      make(chan record, 1024),
		stats:  make(map[string]int),
		ticker: time.NewTicker(interval),
	}

	go s.run()
	return s
}

func (s *Stats) run() {
	for {
		select {
		case record := <-s.c: // received stats
			_, ok := s.stats[record.name]
			if !ok {
				s.stats[record.name] = 0
			}
			s.stats[record.name] += record.delta

		case <-s.ticker.C: // print log every time.Interval
			msgs := []string{}
			for name, val := range s.stats {
				msgs = append(msgs, fmt.Sprintf("%s: %d", name, val))
				s.stats[name] = 0
			}
			log.Printf("Stats: %s", strings.Join(msgs, ", "))
		}
	}

}

func (s *Stats) Record(name string, delta int) {
	s.c <- record{name, delta}
}
