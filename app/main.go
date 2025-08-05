package main

import (
	"bufio"
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/server"
	"io"
	"net"
	"os"
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
	for _, matser := range serv.ConnectedMaster {
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
			out := serv.ProcessCommand(&conn, &cmd)
			if cmd.Name == "REPLCONF" && cmd.Args[0] == "GETACK" {
				conn.Write(out)
			}
			fmt.Println(cmd)

		}
	}

}

func handleConnection(conn net.Conn, serv *server.Server) {
	defer conn.Close()
	defer fmt.Println("connection closed")
	//create a reader source for this connection
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn) // Add buffered writer for better performance
	for {
		cmd, err := server.ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		out := serv.ProcessCommand(&conn, &cmd)
		writer.Write(out)
		writer.Flush()
		if cmd.Name == "PSYNC" {
			comm := server.Command{
				Name: "REPLCONF",
				Args: []string{
					"GETACK", "*",
				},
				IsWritable: false,
				Handle:     nil,
			}
			server.WriteCommand(writer, &comm)
		}
		fmt.Println(cmd)
	}
}
