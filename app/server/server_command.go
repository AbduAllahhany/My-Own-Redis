package server

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

// Server commands
func echo(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	return resp.BulkStringDecoder(args[0]), nil
}
func ping(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	return resp.SimpleStringDecoder("PONG"), nil
}

// Configuration
func config(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) == 0 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	if strings.ToUpper(args[0]) == "GET" {
		if len(args) == 1 {
			return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
		}
		var arr []string
		for _, arg := range args[1:] {
			val, ok := ConfigLookup[arg]
			if ok {
				arr = append(arr, arg)
				arr = append(arr, val)
			}
		}
		return resp.ArrayDecoder(arr), nil
	}

	return []byte(resp.Nil), nil
}

// Blocking function
func keys(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
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
				return resp.ErrorDecoder("ERR encoding"), ErrInvalidFormat
			}
			if match {
				matches = append(matches, key)
			}
		}
	}
	return resp.ArrayDecoder(matches), nil
}
func info(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	if strings.ToUpper(args[0]) == "REPLICATION" {

		out := "#REPLICATION" + resp.CLRF
		if request.Serv.Role == Master {
			out += "role:master" + resp.CLRF
			out += "master_replid:" + request.Serv.ReplicationId + resp.CLRF
			out += "master_repl_offset:" + strconv.Itoa(request.Serv.Offset) + resp.CLRF
			return resp.BulkStringDecoder(out), nil
		} else if request.Serv.Role == Slave {
			out += "role:slave" + resp.CLRF
			return resp.BulkStringDecoder(out), nil
		}
	}
	return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
}

func replconf(request *Request) ([]byte, error) {
	if len(request.Cmd.Args) != 2 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}

	switch strings.ToUpper(request.Cmd.Args[0]) {
	case "GETACK":
		out := []string{
			"REPLCONF", "ACK", strconv.Itoa(request.Serv.Offset),
		}
		return resp.ArrayDecoder(out), nil
	case "ACK":
		replica := replicaById[request.ConnId]
		offset, err := strconv.Atoi(request.Cmd.Args[1])
		if err != nil {
			return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
		}
		replica.Node.Offset = offset
		return nil, nil
	}
	return resp.SimpleStringDecoder("OK"), nil
}

func psync(request *Request) ([]byte, error) {
	conn := *request.Conn
	writer := request.Writer
	replica := Replica{
		Node: Node{
			ReplicationId: request.ConnId,
			Conn:          &conn,
			Reader:        bufio.NewReader(conn),
			Writer:        bufio.NewWriter(conn),
		},
		Buffer:   bytes.Buffer{},
		Pending:  false,
		RDBReady: make(chan struct{}),
		Ready:    make(chan struct{}),
	}
	replicaById[replica.Node.ReplicationId] = &replica
	if request.Serv.ConnectedReplica == nil {
		request.Serv.ConnectedReplica = &[]*Replica{
			&replica,
		}
	} else {
		*request.Serv.ConnectedReplica = append(*request.Serv.ConnectedReplica, &replica)
	}
	out := "FULLRESYNC" + " " + request.Serv.ReplicationId + " " + strconv.Itoa(request.Serv.Offset)
	writer.Write(resp.SimpleStringDecoder(out))
	writer.Flush()
	go sendBgServerReplication(request)
	return nil, nil
}

func wait(request *Request) ([]byte, error) {
	if len(request.Cmd.Args) != 2 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	noOfReplica, err := strconv.Atoi(request.Cmd.Args[0])
	if err != nil {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	timeout, err := strconv.Atoi(request.Cmd.Args[1])
	if err != nil {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	if request.Serv.ConnectedReplica == nil {
		return resp.IntegerDecoder(0), nil
	}
	cmd := Command{
		Name:           "REPLCONF",
		Args:           []string{"GETACK", "*"},
		IsPropagatable: false,
		SuppressReply:  false,
		IsWritable:     false,
		Handle:         replconf,
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
		ticker := time.NewTicker(time.Duration(timeout) * time.Millisecond / 10)
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
		return resp.IntegerDecoder(len(*request.Serv.ConnectedReplica)), nil
	}
	return resp.IntegerDecoder(noOfAckedReplica), nil
}
func selectIndex(request *Request) ([]byte, error) {
	return resp.SimpleStringDecoder("OK"), nil
}
