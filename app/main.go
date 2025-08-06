package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

//Todo
//1)Testing
//2)Implement my own rdb reader
//2)Memory Optimization
//3)Connection pool
//4)Logging

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
	fmt.Println(serv.Role)
	defer l.Close()
	if serv.Role == server.Slave {
		go handleMasterConnection(serv)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn, serv)
	}
}

func handleMasterConnection(serv *server.Server) {
	//buf := make([]byte, 1024)
	matser := serv.ConnectedMaster
	conn := *matser.Conn
	reader := bufio.NewReader(conn)
	for {
		cmd, err := server.ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println(err)
		}
		err = serv.ProcessCommand(&conn, &cmd)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(cmd)

	}
}

func handleConnection(conn net.Conn, serv *server.Server) {
	defer conn.Close()
	defer fmt.Println("connection closed")
	//create a reader source for this connection
	reader := bufio.NewReader(conn)
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
		err = serv.ProcessCommand(&conn, &cmd)
		if err != nil {
			fmt.Println(err)
		}

	}
}
