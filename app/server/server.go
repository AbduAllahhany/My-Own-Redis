package server

import (
	"flag"
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/google/uuid"
	"net"
	"strings"
	"sync"
)

var ConfigLookup map[string]string

const (
	Master = "MASTER"
	Slave  = "SLAVE"
)

type Config struct {
	Dir        string
	DbFilename string
	Port       string
	MasterInfo string
}
type Replica struct {
	Id   string
	Port int
	Ip   net.IP
}
type Server struct {
	Id               string
	Db               engine.DbStore
	Configuration    Config
	Listener         net.Listener
	Role             string
	ConnectedReplica []Replica
}

func NewServer(config Config) (*Server, error) {
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
	host := parts[0]
	port := parts[1]
	address := net.JoinHostPort(host, port)
	_, err = net.ResolveTCPAddr("tcp", address)
	role := Master
	if err == nil {
		role = Slave
	}

	serv := Server{
		Id:               uuid.New().String(),
		Db:               db,
		Configuration:    config,
		Listener:         l,
		Role:             role,
		ConnectedReplica: nil,
	}

	return &serv, nil
}

func NewConfiguration() Config {
	dir := flag.String("dir", "/tmp", "Directory path for data storage")
	dbfilename := flag.String("dbfilename", "dump.rdb", "Database filename")
	port := flag.String("port", "6379", "Port")
	replica := flag.String("replicaof", "Bad Address", "Is Replica")
	flag.Parse()
	return Config{
		Dir:        *dir,
		DbFilename: *dbfilename,
		Port:       *port,
		MasterInfo: *replica,
	}
}

func configToMap(config Config) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
	}
}
