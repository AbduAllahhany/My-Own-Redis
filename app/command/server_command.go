package command

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/server"
	"path/filepath"
	"strings"
)

// Server commands
func Echo(serv *server.Server, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.BulkStringDecoder(args[0])
}
func Ping(serv *server.Server, args []string) []byte {
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.SimpleStringDecoder("PONG")
}

// Config
func Config(serv *server.Server, args []string) []byte {
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "GET" {
		return configGet(serv, args[1:])
	}

	return []byte(resp.Nil)
}
func configGet(serv *server.Server, args []string) []byte {
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	var arr []string
	for _, arg := range args {
		val, ok := server.ConfigLookup[arg]
		if ok {
			arr = append(arr, arg)
			arr = append(arr, val)
		}
	}
	return resp.ArrayDecoder(arr)
}

// Blocking function
func Keys(serv *server.Server, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	store := serv.Db
	dict := store.Dict
	mu := store.Mu
	mu.Lock()
	defer mu.Unlock()
	pattern := args[0]
	var matches []string
	for key, _ := range dict {
		match, err := filepath.Match(pattern, key)
		if err != nil {
			return resp.ErrorDecoder("ERR encoding")
		}
		if match {
			matches = append(matches, key)
		}
	}
	return resp.ArrayDecoder(matches)
}
func Info(serv *server.Server, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "REPLICATION" {

		out := "#REPLICATION" + resp.CLRF
		if serv.Role == server.Master {
			out += "role:master" + resp.CLRF
			return resp.BulkStringDecoder(out)
		} else if serv.Role == server.Slave {
			out += "role:slave" + resp.CLRF
			return resp.BulkStringDecoder(out)
		}
	}
	return resp.ErrorDecoder("ERR syntax error")
}
