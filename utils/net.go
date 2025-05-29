package utils

import (
	"bufio"
	"io"
	"net"
	"strings"
)

func SendMessage(conn net.Conn, msg string) {
	conn.Write([]byte(msg))
}

func ReadFromConn(conn net.Conn) string {
	reader := bufio.NewReader(conn)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			conn.Close()
		}
		return ""
	}
	return strings.TrimSpace(input)
}
