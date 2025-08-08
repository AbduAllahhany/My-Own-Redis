package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/server"
	"github.com/codecrafters-io/redis-starter-go/app/utils"
)

// Todo
// 1)Testing
// 2)Implement my own rdb reader
// 2)Memory Optimization
// 3)Connection pool
// 4)Logging
// 5)Read more about offset tracking and implement this
// 6)Improve readCommand function
// 7)server implement reader and writer
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
	//create connectionId for this conenction
	connId := utils.GenerateID()

	for {
		cmd, err := server.ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		request := server.Request{
			Serv:   serv,
			Conn:   connection,
			Reader: reader,
			Writer: writer,
			Cmd:    &cmd,
			ConnId: connId,
		}
		out, err := server.ProcessCommand(&request)
		if err != nil {
			fmt.Println(err)
		}
		writer.Write(out)
		writer.Flush()
	}
}
