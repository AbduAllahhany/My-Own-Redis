package server

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/codecrafters-io/redis-starter-go/app/utils"
)

var replicaById map[string]*Replica
var ConfigLookup map[string]string

const (
	Master = "MASTER"
	Slave  = "SLAVE"
)

type Request struct {
	Serv   *Server
	Conn   *net.Conn
	Reader *bufio.Reader
	Writer *bufio.Writer
	Cmd    *Command
	ConnId string
}
type Configuration struct {
	Dir        string
	DbFilename string
	Port       string
	MasterInfo string
}
type Node struct {
	ReplicationId string
	Offset        int
	Conn          *net.Conn
	Reader        *bufio.Reader
	Writer        *bufio.Writer
}

func (r Replica) Write(p []byte) (n int, err error) {
	writer := *r.Node.Writer
	n, err = writer.Write(p)
	writer.Flush()
	return n, err
}

type Server struct {
	ReplicationId    string
	Db               *engine.DbStore
	Configuration    Configuration
	Listener         net.Listener
	Role             string
	Offset           int
	ConnectedReplica *[]*Replica
	ConnectedMaster  *Node
}

func (serv Server) Run() {

	l := serv.Listener
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(&conn, &serv)
	}
}

// type shitt
func NewServer(config Configuration) (*Server, error) {
	replicaById = make(map[string]*Replica)
	InitCommands()
	ConfigLookup = configToMap(&config)
	//create data store instance
	path := config.Dir + "/" + config.DbFilename
	dict := rdb.Encode(path)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}
	serverAddress := net.JoinHostPort("localhost", config.Port)

	//set up networking
	l, err := net.Listen("tcp", serverAddress)
	if err != nil {
		return nil, err
	}

	serv := Server{
		ReplicationId:    utils.GenerateID(),
		Db:               &db,
		Configuration:    config,
		Listener:         l,
		Role:             Master,
		Offset:           0,
		ConnectedReplica: nil,
	}
	StartReplicationFlusher()
	go connectToMaster(&serv, &config)
	return &serv, nil
}

func NewConfiguration() Configuration {
	dir := flag.String("dir", "/tmp", "Directory path for data storage")
	dbfilename := flag.String("dbfilename", "dump.rdb", "Database filename")
	port := flag.String("port", "6379", "Port")
	replica := flag.String("replicaof", "nil", "Is Replica")
	flag.Parse()
	confg := Configuration{
		Dir:        *dir,
		DbFilename: *dbfilename,
		Port:       *port,
		MasterInfo: *replica,
	}
	return confg
}

func configToMap(config *Configuration) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
		"port":       config.Port,
	}
}

// kanye west reference
func NewSlave(serv *Server) error {
	pingCmd := Command{
		Name:   "PING",
		Args:   nil,
		Handle: ping,
	}
	repliconfPortArgs := []string{
		"listening-port",
		serv.Configuration.Port,
	}
	repliconfPortCmd := Command{
		Name:   "REPLCONF",
		Args:   repliconfPortArgs,
		Handle: replconf,
	}

	repliconfCapArgs := []string{
		"capa",
		"psync2",
	}
	repliconfCapCmd := Command{
		Name:   "REPLCONF",
		Args:   repliconfCapArgs,
		Handle: replconf,
	}
	psyncCmdArgs := []string{
		"?",
		"-1",
	}
	psyncCmd := Command{
		Name:   "PSYNC",
		Args:   psyncCmdArgs,
		Handle: nil,
	}
	//buf := make([]byte, 1024)
	node := serv.ConnectedMaster
	writer := node.Writer
	reader := node.Reader
	err := WriteCommand(writer, &pingCmd)
	if err != nil {
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	err = WriteCommand(writer, &repliconfPortCmd)
	if err != nil {
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	err = WriteCommand(writer, &repliconfCapCmd)
	if err != nil {
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	err = WriteCommand(writer, &psyncCmd)
	if err != nil {
		return err
	}
	line, err := reader.ReadString('\n')

	if err != nil {
		return err
	}
	_, err = readRDBSnapshot(reader)
	if err != nil {
		return err
	}
	line = strings.TrimSuffix(line, "\r\n")
	parts := strings.Split(line, " ")
	offsetStr := parts[2]
	offset, err := strconv.Atoi(offsetStr)
	serv.Offset = offset
	go handleMasterConnection(serv)
	return nil
}
func handleConnection(connection *net.Conn, serv *Server) {
	conn := *connection
	defer conn.Close()
	defer fmt.Println("connection closed")
	//create a reader source for this connection
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	//create connectionId for this conenction
	connId := utils.GenerateID()

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
			Conn:   connection,
			Reader: reader,
			Writer: writer,
			Cmd:    &cmd,
			ConnId: connId,
		}
		out, err := ProcessCommand(&request)
		if err != nil {
			fmt.Println(err)
		}
		writer.Write(out)
		writer.Flush()
	}
}
