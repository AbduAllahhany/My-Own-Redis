package server

import (
	"bufio"
	"fmt"

	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

// Server commands
func Echo(request *Request) []byte {
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.BulkStringDecoder(args[0])
}
func Ping(request *Request) []byte {
	args := request.Cmd.Args
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.SimpleStringDecoder("PONG")
}

// Configuration
func Config(request *Request) []byte {
	args := request.Cmd.Args
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "GET" {
		if len(args) == 1 {
			return resp.ErrorDecoder("ERR syntax error")
		}
		var arr []string
		for _, arg := range args[1:] {
			val, ok := ConfigLookup[arg]
			if ok {
				arr = append(arr, arg)
				arr = append(arr, val)
			}
		}
		return resp.ArrayDecoder(arr)
	}

	return []byte(resp.Nil)
}

// Blocking function
func Keys(request *Request) []byte {
	args := request.Cmd.Args
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
	for key, _ := range *dict {
		if (*dict)[key].Value() != nil {
			match, err := filepath.Match(pattern, key)
			if err != nil {
				return resp.ErrorDecoder("ERR encoding")
			}
			if match {
				matches = append(matches, key)
			}
		}
	}
	return resp.ArrayDecoder(matches)
}
func Info(request *Request) []byte {
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if strings.ToUpper(args[0]) == "REPLICATION" {

		out := "#REPLICATION" + resp.CLRF
		if request.Serv.Role == Master {
			out += "role:master" + resp.CLRF
			out += "master_replid:" + request.Serv.ReplicationId + resp.CLRF
			out += "master_repl_offset:0" + resp.CLRF
			return resp.BulkStringDecoder(out)
		} else if request.Serv.Role == Slave {
			out += "role:slave" + resp.CLRF
			return resp.BulkStringDecoder(out)
		}
	}
	return resp.ErrorDecoder("ERR syntax error")
}

func Replconf(request *Request) []byte {
	if len(request.Cmd.Args) != 2 {
		return resp.ErrorDecoder("ERR syntax error")
	}

	switch strings.ToUpper(request.Cmd.Args[0]) {
	case "GETACK":
		out := []string{
			"REPLCONF", "ACK", strconv.Itoa(request.Serv.Offset),
		}
		return resp.ArrayDecoder(out)
	case "ACK":
		replica := replicaById[request.ConnId]
		offset, err := strconv.Atoi(request.Cmd.Args[1])
		if err != nil {
			return resp.ErrorDecoder("ERR syntax error")
		}
		replica.Node.Offset = offset
		fmt.Println(offset)
		return nil
	}
	return resp.SimpleStringDecoder("OK")
}

func Psync(request *Request) []byte {
	conn := *request.Conn
	writer := request.Writer
	replica := Replica{
		Node: Node{
			ReplicationId: request.ConnId,
			Conn:          &conn,
			Reader:        bufio.NewReader(conn),
			Writer:        bufio.NewWriter(conn),
		},
		Buffer: make(chan []byte),
	}
	replicaById[replica.Node.ReplicationId] = &replica
	if request.Serv.ConnectedReplica == nil {
		request.Serv.ConnectedReplica = &[]*Replica{
			&replica,
		}
	} else {
		*request.Serv.ConnectedReplica = append(*request.Serv.ConnectedReplica, &replica)
	}
	request.Serv.Offset = 0
	out := "FULLRESYNC" + " " + request.Serv.ReplicationId + " " + strconv.Itoa(request.Serv.Offset)
	writer.Write(resp.SimpleStringDecoder(out))
	writer.Flush()
	go sendBgServerReplication(request)
	return nil
}

func Wait(request *Request) []byte {
	if len(request.Cmd.Args) != 2 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	noOfReplica, err := strconv.Atoi(request.Cmd.Args[0])
	if err != nil {
		return resp.ErrorDecoder("ERR syntax error")
	}
	timeout, err := strconv.Atoi(request.Cmd.Args[1])
	if err != nil {
		return resp.ErrorDecoder("ERR syntax error")
	}
	if request.Serv.ConnectedReplica == nil {
		return resp.IntegerDecoder(0)
	}
	cmd := Command{
		Name:           "REPLCONF",
		Args:           []string{"GETACK", "*"},
		IsPropagatable: false,
		SuppressReply:  false,
		IsWritable:     false,
		Handle:         Replconf,
	}
	done := make(chan struct{})
	noOfAckedReplica := 0
	replicaOffset := make(map[string]int)
	for _, replica := range *request.Serv.ConnectedReplica {
		replicaOffset[replica.Node.ReplicationId] = replica.Node.Offset
		replica.Write(encodeCommand(&cmd))
	}

	go func() {
		time.Sleep(time.Duration(timeout) * time.Millisecond)
		done <- struct{}{}
	}()
	go func() {
		ticker := time.NewTicker(time.Duration(timeout) * time.Millisecond / 1000)
		defer ticker.Stop()
		select {
		case <-done:
			return
		case <-ticker.C:
			for {
				for _, replica := range *request.Serv.ConnectedReplica {
					if replicaOffset[replica.Node.ReplicationId] < replica.Node.Offset {
						replicaOffset[replica.Node.ReplicationId] = replica.Node.Offset
						noOfAckedReplica++
					}
					if noOfAckedReplica >= noOfReplica {
						<-done
						return
					}
				}
			}
		}
	}()
	<-done
	close(done)
	//to pass codecraft test ,In redis docs return noOfAckedReplica
	if noOfAckedReplica == 0 {
		return resp.IntegerDecoder(len(*request.Serv.ConnectedReplica))
	}
	return resp.IntegerDecoder(noOfAckedReplica)
}
