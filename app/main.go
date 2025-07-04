package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

const MaxBulkStringSize = 512 * 1024 * 1024 // 512MB limit

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
		line, err := reader.ReadString('\n')
		var command []string

		if err != nil {
			fmt.Println(err)
			return
		}

		if strings.HasSuffix(line, "\r\n") {
			matches := numberRegex.FindAllString(line, -1)
			if len(matches) <= 0 {
				return
			}
			noOfLines, _ := strconv.Atoi(matches[0])
			for range noOfLines {
				str, err := readBulkString(reader)
				if err != nil {
					fmt.Println(err)
					command = nil
					return
				}
				command = append(command, str)
			}
			fmt.Println(command)
			conn.Write([]byte("+PONG\r\n"))
		}
	}
}
func readBulkString(reader *bufio.Reader) (string, error) {
	var command string

	//read the no of bytes
	line, err := reader.ReadString('\n')
	if err != nil || line[0] != '$' {
		fmt.Println(err)
		return "", errors.New("Invalid Format")
	}

	if strings.HasSuffix(line, "\r\n") {
		matches := numberRegex.FindAllString(line, -1)
		if len(matches) <= 0 {
			return "", errors.New("Invalid Format")
		}
		noOfBytes, err := strconv.Atoi(matches[0])
		if err != nil {
			return "", errors.New("Invalid bulk string length")
		}
		// Add bounds checking here
		if noOfBytes < 0 || noOfBytes > MaxBulkStringSize {
			return "", errors.New("Bulk string too large")
		}
		// the two bytes of \r\n
		noOfBytes += 2

		//read command
		buf := make([]byte, noOfBytes)
		n, err := reader.Read(buf)
		if err != nil || n != noOfBytes {
			fmt.Println(err)
			return "", errors.New("Invalid Format")
		}
		line = string(buf)
		if strings.HasSuffix(line, "\r\n") {
			command = line[:len(line)-2]
		}
	}
	return command, nil
}

var numberRegex = regexp.MustCompile(`\d+`)
