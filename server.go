package multiplex

import (
	"bufio"
	"io"
)

func HandleConn(conn io.ReadWriteCloser) {
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
