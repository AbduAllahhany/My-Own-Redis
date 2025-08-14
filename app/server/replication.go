package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/rdb"
)

type Replica struct {
	Node     Node
	Buffer   bytes.Buffer
	State    string
	Pending  bool //for patching commands
	RDBReady chan struct{}
	Ready    chan struct{} //for sending command after rdb
}

var replicasPendingWrite = make(chan *Replica)

func readRDBSnapshot(reader *bufio.Reader) ([]byte, error) {
	// Read $<len>\r\n
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	lengthStr := strings.TrimSpace(line)[1:] // Remove $ and \r\n
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, err
	}

	if length == -1 {
		return nil, nil
	}

	// Read RDB data
	data := make([]byte, length)
	_, err = io.ReadFull(reader, data)
	return data, err
}
func sendBgServerReplication(request *Request) {
	replica := replicaById[request.ConnId]
	writer := request.Writer
	rdbData, _ := rdb.GenerateRDBBinary(request.Serv.Db.Dict)
	lengthLine := fmt.Sprintf("$%d\r\n", len(rdbData))
	writer.Write([]byte(lengthLine)) // send bulk string header
	writer.Write(rdbData)
	writer.Flush()
	close(replica.Ready)
}
func StartReplicationFlusher() {
	go func() {
		for replica := range replicasPendingWrite {
			<-replica.Ready
			if replica.Buffer.Len() > 0 {
				data := replica.Buffer.Bytes()
				n, err := replica.Write(data)
				if err != nil {
					continue
				}
				if n < len(data) {
					replica.Buffer.Next(n)
				} else {
					replica.Buffer.Reset()
				}
			}
			replica.Pending = false

		}
	}()
}
func queueReplica(replica *Replica) {
	if !replica.Pending {
		replica.Pending = true
		replicasPendingWrite <- replica
	}
}
func connectToMaster(serv *Server, config *Configuration) {
	parts := strings.Split(config.MasterInfo, " ")
	if len(parts) != 2 {
		return
	}
	masterIp := parts[0]
	masterPort := parts[1]
	address := net.JoinHostPort(masterIp, masterPort)
	_, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return
	}
	serv.Role = Slave
	var masterConn net.Conn
	for {
		masterConn, err = net.Dial("tcp", address)
		if err == nil {
			break
		}
	}
	serv.ConnectedMaster = &Node{
		Conn:   &masterConn,
		Reader: bufio.NewReader(masterConn),
		Writer: bufio.NewWriter(masterConn),
	}
	NewSlave(serv)
}
func handleMasterConnection(serv *Server) {
	master := serv.ConnectedMaster
	conn := master.Conn
	reader := master.Reader
	writer := master.Writer
	fmt.Println("i am in handle master connection")
	for {
		cmd, err := ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		request := Request{
			Serv:   serv,
			Conn:   conn,
			Reader: reader,
			Writer: writer,
			Cmd:    &cmd,
		}
		out, err := ProcessCommand(&request)
		serv.Offset += len(encodeCommand(&cmd))
		fmt.Println(serv.Offset)
		if err != nil {
			fmt.Println(err)
		}
		if !cmd.SuppressReply {
			writer.Write(out)
			writer.Flush()
		}
	}
}
