package server

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

// Server commands
func Echo(request *Request) []byte {
	args := request.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.BulkStringDecoder(args[0])
}
func Ping(request *Request) []byte {
	args := request.Args
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.SimpleStringDecoder("PONG")
}

// Configuration
func Config(request *Request) []byte {
	args := request.Args
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "GET" {
		req := Request{
			Serv: request.Serv,
			Args: args[1:],
			Conn: request.Conn,
		}
		return configGet(&req)
	}

	return []byte(resp.Nil)
}
func configGet(request *Request) []byte {
	args := request.Args

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
func Keys(request *Request) []byte {
	args := request.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	store := request.Serv.Db
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
func Info(request *Request) []byte {
	args := request.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "REPLICATION" {

		out := "#REPLICATION" + resp.CLRF
		if request.Serv.Role == Master {
			out += "role:master" + resp.CLRF
			out += "master_replid:" + request.Serv.Id + resp.CLRF
			out += "master_repl_offset:0" + resp.CLRF
			return resp.BulkStringDecoder(out)
		} else if request.Serv.Role == Slave {
			out += "role:slave" + resp.CLRF
			return resp.BulkStringDecoder(out)
		}
	}
	return resp.ErrorDecoder("ERR syntax error")
}

func Replconfg(request *Request) []byte {
	if strings.ToUpper(request.Args[0]) == "GETACK" {
		req := Request{
			Serv: request.Serv,
			Args: request.Args[1:],
			Conn: request.Conn,
		}
		return replconfgGetAck(&req)
	}
	return resp.SimpleStringDecoder("OK")
}
func replconfgGetAck(request *Request) []byte {
	out := []string{
		"REPLCONF", "ACK", "0",
	}
	return resp.ArrayDecoder(out)
}
func Psync(request *Request) []byte {
	conn := *request.Conn
	replica := Node{
		Id:   generateID(),
		Conn: &conn,
	}
	if request.Serv.ConnectedReplica == nil {
		request.Serv.ConnectedReplica = []Node{
			replica,
		}
	} else {
		request.Serv.ConnectedReplica = append(request.Serv.ConnectedReplica, replica)
	}
	out := "FULLRESYNC" + " " + request.Serv.Id + " " + strconv.Itoa(request.Serv.offset)
	go sendbgserverReplication(request)
	return resp.SimpleStringDecoder(out)
}
func sendbgserverReplication(request *Request) {
	conn := *request.Conn
	res, _ := rdb.GenerateRDBBinary(request.Serv.Db.Dict)
	out := resp.BulkStringDecoder(string(res))
	conn.Write(out[:len(out)-2])
}
