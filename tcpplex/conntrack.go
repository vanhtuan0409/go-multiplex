package main

import "errors"

var (
	ErrStreamExisted       = errors.New("stream existed")
	currStreamID     int16 = 0
)

type Conntrack struct {
	m map[int16]*Stream
}

func NewConntrackTable() *Conntrack {
	return &Conntrack{
		m: make(map[int16]*Stream),
	}
}

func (t *Conntrack) NextID() int16 {
	currStreamID += 1
	return currStreamID
}

func (t *Conntrack) Register(s *Stream) {
	if _, ok := t.m[s.id]; ok {
		return
	}

	// register
	t.m[s.id] = s

	// deregister on close
	s.onClose = func() {
		delete(t.m, s.id)
		if s.onClose != nil {
			s.onClose()
		}
	}
}

// Destroy passive destroy a stream when remote end was closed
func (t *Conntrack) Destroy(id int16) {
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

func (t *Conntrack) Lookup(id int16) *Stream {
	s, ok := t.m[id]
	if !ok {
		return nil
	}
	if s.closed {
		return nil
	}
	return s
}
