package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/command"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit


func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()
	for true {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}
func handleConnection(conn net.Conn) {
	defer conn.Close()
	defer fmt.Println("connection closed")
	//create a reader source for this connection

	reader := bufio.NewReader(conn)
	for {
		command, err := command.Read(reader)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(command)
		command.Proccess(conn, &command)
	}
}
