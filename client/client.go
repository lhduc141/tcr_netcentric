package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		fmt.Println("❌ Cannot connect to server:", err)
		return
	}
	defer conn.Close()

	go listen(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		text, _ := reader.ReadString('\n')
		fmt.Fprintf(conn, text)
	}
}

func listen(conn net.Conn) {
	for {
		message := make([]byte, 1024)
		len, _ := conn.Read(message)
		fmt.Print(string(message[:len]))
	}
}
