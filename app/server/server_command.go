package server

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"path/filepath"
	"strings"
)

// Server commands
func Echo(serv *Server, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.BulkStringDecoder(args[0])
}
func Ping(serv *Server, args []string) []byte {
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.SimpleStringDecoder("PONG")
}

// Configuration
func Config(serv *Server, args []string) []byte {
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "GET" {
		return configGet(serv, args[1:])
	}

	return []byte(resp.Nil)
}
func configGet(serv *Server, args []string) []byte {
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	var arr []string
	for _, arg := range args {
		val, ok := ConfigLookup[arg]
		if ok {
			arr = append(arr, arg)
			arr = append(arr, val)
		}
	}
	return resp.ArrayDecoder(arr)
}

// Blocking function
func Keys(serv *Server, args []string) []byte {
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
func Info(serv *Server, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "REPLICATION" {

		out := "#REPLICATION" + resp.CLRF
		if serv.Role == Master {
			out += "role:master" + resp.CLRF
			out += "master_replid:" + serv.Id + resp.CLRF
			out += "master_repl_offset:0" + resp.CLRF
			return resp.BulkStringDecoder(out)
		} else if serv.Role == Slave {
			out += "role:slave" + resp.CLRF
			return resp.BulkStringDecoder(out)
		}
	}
	return resp.ErrorDecoder("ERR syntax error")
}

func Replconfg(serv *Server, args []string) []byte {
	if serv.ConnectedReplica == nil {
		serv.ConnectedReplica = []Node{
			{
				Port: "",
				Ip:   "",
			},
		}
	}
	return resp.SimpleStringDecoder("OK")
}
