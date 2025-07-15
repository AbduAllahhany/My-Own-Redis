package server

import (
	"flag"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"net"
	"sync"
)

type Config struct {
	Dir        string
	DbFilename string
	Port       string
}

var ConfigLookup map[string]string

type Server struct {
	Db            engine.DbStore
	Configuration Config
	Listener      net.Listener
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
	serv := Server{
		Db:            db,
		Configuration: config,
		Listener:      l,
	}

	return &serv, nil
}

func NewConfiguration() Config {
	dir := flag.String("dir", "/tmp", "Directory path for data storage")
	dbfilename := flag.String("dbfilename", "dump.rdb", "Database filename")
	port := flag.String("port", "6379", "Port")
	flag.Parse()
	return Config{
		Dir:        *dir,
		DbFilename: *dbfilename,
		Port:       *port,
	}
}

func configToMap(config Config) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
	}
}
