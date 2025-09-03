package main

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

//priority Todo
//buffer size --> reading long keys cause an issues

// Todo
// 1)Testing
// 2)Implement my own rdb reader
// 2)Memory Optimization
// 3)Connection pool
// 4)Logging
// 5)Read more about offset tracking and implement this
// 6)Improve readCommand function
// 7)server implement reader and writer
// 8) implement my own radix tree
// 9)Avoid unnecessary string ↔ byte conversions.
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")
	cfg := server.NewConfiguration()
	serv, err := server.NewServer(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	serv.Run()
}
