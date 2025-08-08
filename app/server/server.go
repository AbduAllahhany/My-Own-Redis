package server

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
)

//todo
//refactor this shittttttttttttttttttttttttttttttttttttttttttttttt
//implement proper master ,slave ???
//command ?????????????????????

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
type Replica struct {
	Node    Node
	Buffer  chan []byte
	RDBDone chan struct{}
	State   string
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
	ConnectedReplica *[]Replica
	ConnectedMaster  *Node
}

// type shitt
func NewServer(config Configuration) (*Server, error) {
	initCommands()
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
		ReplicationId:    generateID(),
		Db:               &db,
		Configuration:    config,
		Listener:         l,
		Role:             Master,
		Offset:           0,
		ConnectedReplica: nil,
	}
	go connectToMaster(&serv, &config)
	return &serv, nil
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
	fmt.Println(serv.Role)
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

// helper
func configToMap(config *Configuration) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
		"port":       config.Port,
	}
}
func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 40
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return ""
		}
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// kanye west reference
func NewSlave(serv *Server) error {
	pingCmd := Command{
		Name:   "PING",
		Args:   nil,
		Handle: Ping,
	}
	repliconfPortArgs := []string{
		"listening-port",
		serv.Configuration.Port,
	}
	repliconfPortCmd := Command{
		Name:   "REPLCONF",
		Args:   repliconfPortArgs,
		Handle: Replconfg,
	}

	repliconfCapArgs := []string{
		"capa",
		"psync2",
	}
	repliconfCapCmd := Command{
		Name:   "REPLCONF",
		Args:   repliconfCapArgs,
		Handle: Replconfg,
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
		fmt.Println(err)
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &repliconfPortCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &repliconfCapCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &psyncCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = readRDBSnapshot(reader)
	if err != nil {
		fmt.Println(err)
		return err
	}
	go handleMasterConnection(serv)
	return nil
}
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
	writer := request.Writer
	rdbDone := make(chan struct{})
	go func() {
		rdbData, _ := rdb.GenerateRDBBinary(request.Serv.Db.Dict)
		lengthLine := fmt.Sprintf("$%d\r\n", len(rdbData))
		writer.Write([]byte(lengthLine)) // send bulk string header
		writer.Write(rdbData)
		writer.Flush()
		close(rdbDone)
	}()
	go func() {
		<-rdbDone
		writeBufferToReplica(request)
	}()
}

// to be refactored as event loop or smth
func writeBufferToReplica(request *Request) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, replica := range *request.Serv.ConnectedReplica {
				select {
				case buf := <-replica.Buffer:
					replica.Node.Offset += len(buf)
					replica.Write(buf)
				default:
					// no data available
				}
			}
		}
	}
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
		}
		request := Request{
			Serv:   serv,
			Conn:   conn,
			Reader: reader,
			Writer: writer,
			Cmd:    &cmd,
		}
		out, err := ProcessCommand(&request)
		if err != nil {
			fmt.Println(err)
		}
		if !cmd.SuppressReply {
			writer.Write(out)
			writer.Flush()
		}
		fmt.Println("from handle master", cmd)
	}
}

func WriteToSlaveBuffer(replica *Replica, cmd *Command) {
	res := encodeCommand(cmd)
	replica.Buffer <- res
}
