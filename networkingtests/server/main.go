package main

import (
	"fmt"
	"net"
)

func main() {
	listen, err := net.Listen("tcp", "localhost:7777")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	_, err := conn.Write([]byte("Hello World!"))
	if err != nil {
		fmt.Println("Could not write to conn: ", err)
		return
	}
}
