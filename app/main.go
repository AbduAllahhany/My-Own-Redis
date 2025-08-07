package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

// Todo
// 1)Testing
// 2)Implement my own rdb reader
// 2)Memory Optimization
// 3)Connection pool
// 4)Logging
// 4)instead of go routine can use low level epoll
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	cfg := server.NewConfiguration()
	serv, err := server.NewServer(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	l := serv.Listener
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(&conn, serv)
	}
}

func handleConnection(connection *net.Conn, serv *server.Server) {
	conn := *connection
	defer conn.Close()
	defer fmt.Println("connection closed")
	//create a reader source for this connection
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		cmd, err := server.ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(cmd)
		err = serv.ProcessCommand(&conn, reader, writer, &cmd)
		if err != nil {
			fmt.Println(err)
		}

	}
}
