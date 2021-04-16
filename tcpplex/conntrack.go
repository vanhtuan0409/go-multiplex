package main

import "errors"

var (
	ErrStreamExisted = errors.New("stream existed")
	currStreamID     = 0
)

type Conntrack struct {
	m map[int]*Stream
}

func NewConntrackTable() *Conntrack {
	return &Conntrack{
		m: make(map[int]*Stream),
	}
}

func (t *Conntrack) NextID() int {
	currStreamID += 1
	return currStreamID
}

func (t *Conntrack) Register(s *Stream) {
	if _, ok := t.m[s.id]; ok {
		return
	}

	// deregister on close
	s.onClose = func() {
		delete(t.m, s.id)
		if s.onClose != nil {
			s.onClose()
		}
	}
}

// Destroy passive destroy a stream when remote end was closed
func (t *Conntrack) Destroy(id int) {
	s, ok := t.m[id]
	if !ok {
		return
	}
	s.destroy() // auto trigger onClose in Stream
}

func (t *Conntrack) DestroyAll() {
	for id := range t.m {
		t.Destroy(id)
	}
}

func (t *Conntrack) Lookup(id int) *Stream {
	s, ok := t.m[id]
	if !ok {
		return nil
	}
	if s.closed {
		return nil
	}
	return s
}
