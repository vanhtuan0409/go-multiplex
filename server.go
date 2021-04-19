package multiplex

import (
	"bufio"
	"log"
)

type Listener interface {
	Run()
	Accept() (Conn, error)
}

type Server struct {
	L Listener
}

func (s *Server) handleConn(conn Conn) {
	defer conn.Close()

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			break
		}
		if _, err := conn.Write(line); err != nil {
			break
		}
	}

}

func (s *Server) Run() {
	go s.L.Run()
	for {
		conn, err := s.L.Accept()
		if err != nil {
			log.Printf("Failed to accept connection. ERR: %+v", err)
			break
		}

		go s.handleConn(conn)
	}
}
