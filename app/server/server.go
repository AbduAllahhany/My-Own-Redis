package server

import (
	"flag"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"sync"
)

type Config struct {
	Dir        string
	DbFilename string
}

var ConfigLookup map[string]string

type Server struct {
	Db            engine.DbStore
	Configuration Config
}

func NewServer(config Config) *Server {
	//create data store instance
	db := engine.DbStore{
		Dict: make(map[string]engine.RedisObj),
		Mu:   sync.RWMutex{},
	}
	ConfigLookup = configToMap(config)
	serv := Server{
		Db:            db,
		Configuration: config,
	}

	return &serv
}

func NewConfiguration() Config {
	dir := flag.String("dir", "/tmp", "Directory path for data storage")
	dbfilename := flag.String("dbfilename", "dump.rdb", "Database filename")

	flag.Parse()

	return Config{
		Dir:        *dir,
		DbFilename: *dbfilename,
	}
}

func configToMap(config Config) map[string]string {
	return map[string]string{
		"dir":        config.Dir,
		"dbfilename": config.DbFilename,
	}
}
