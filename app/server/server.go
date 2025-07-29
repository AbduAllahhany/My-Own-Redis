package server

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"math/big"
	"net"
	"strings"
	"sync"
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

type Configuration struct {
	Dir        string
	DbFilename string
	Port       string
	MasterInfo string
}
type Node struct {
	Id   string
	Port string
	Ip   string
}
type Server struct {
	Id               string
	Db               engine.DbStore
	Configuration    Configuration
	Listener         net.Listener
	Role             string
	ConnectedReplica []Node
	ConnectedMaster  []Node
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
	_, err = net.ResolveTCPAddr("tcp", address)
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
		ConnectedMaster:  nil,
	}
	if role == Slave {
		serv.ConnectedMaster = []Node{
			{
				Port: masterPort,
				Ip:   masterIp,
			},
		}
		slaveInit(&serv)
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

func slaveInit(serv *Server) error {
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

	/*repliconfCapArgs := []string{
		"capa",
		"psync2",
	}
	repliconfCapCmd := Command{
		Name:   "REPLCONF",
		Args:   repliconfCapArgs,
		Handle: Replconfg,
	}
	*/
	for _, node := range serv.ConnectedMaster {

		conn, err := net.Dial("tcp", node.Ip+":"+node.Port)
		if err != nil {
			return err
		}
		defer conn.Close()
		writer := bufio.NewWriter(conn)
		err = WriteCommand(writer, &pingCmd)
		if err != nil {
			fmt.Println(err)
		}
		err = WriteCommand(writer, &repliconfPortCmd)
		if err != nil {
			fmt.Println(err)
		}
		//err = WriteCommand(writer, &repliconfCapCmd)
		if err != nil {
			fmt.Println(err)
		}
	}
	return nil
}
