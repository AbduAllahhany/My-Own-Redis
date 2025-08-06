package server

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"

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
	Serv *Server
	Args []string
	//metadata
	Conn *net.Conn
}
type Configuration struct {
	Dir        string
	DbFilename string
	Port       string
	MasterInfo string
}
type Node struct {
	Id   string
	Conn *net.Conn
}
type Server struct {
	Id               string
	Db               engine.DbStore
	Configuration    Configuration
	Listener         net.Listener
	Role             string
	offset           int
	ConnectedReplica []Node
	ConnectedMaster  Node
}

// type shitt
func NewServer(config Configuration) (*Server, error) {
	//create data store instance
	path := config.Dir + "/" + config.DbFilename
	dict := rdb.Encode(path)
	db := engine.DbStore{
		Dict: dict,
		Mu:   sync.RWMutex{},
	}
	ConfigLookup = configToMap(config)
	l, err := net.Listen("tcp", "0.0.0.0:"+config.Port)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(config.MasterInfo, " ")
	if len(parts) != 2 {
		fmt.Println("Invalid replica address. Expected 'host port'")
	}
	masterIp := parts[0]
	masterPort := parts[1]
	address := net.JoinHostPort(masterIp, masterPort)
	masterConn, err := net.Dial("tcp", address)

	role := Master
	if err == nil {
		role = Slave
	}

	serv := Server{
		Id:               generateID(),
		Db:               db,
		Configuration:    config,
		Listener:         l,
		Role:             role,
		ConnectedReplica: nil,
	}
	if role == Slave {
		serv.ConnectedMaster = Node{
			Conn: &masterConn,
		}
		NewSlave(&serv)
	}
	return &serv, nil
}

func NewConfiguration() Configuration {
	dir := flag.String("dir", "/tmp", "Directory path for data storage")
	dbfilename := flag.String("dbfilename", "dump.rdb", "Database filename")
	port := flag.String("port", "6379", "Port")
	replica := flag.String("replicaof", "Bad Address", "Is Replica")
	flag.Parse()
	return Configuration{
		Dir:        *dir,
		DbFilename: *dbfilename,
		Port:       *port,
		MasterInfo: *replica,
	}
}

// helper
func configToMap(config Configuration) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
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
	buf := make([]byte, 1024)
	node := serv.ConnectedMaster
	conn := *node.Conn
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	err := WriteCommand(writer, &pingCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &repliconfPortCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &repliconfCapCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = WriteCommand(writer, &psyncCmd)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = reader.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, err = conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(buf))

	return nil
}
