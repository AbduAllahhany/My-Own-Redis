package server

import (
	"flag"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/rdb"
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
	path := config.Dir + "/" + config.DbFilename
	dict := rdb.Encode(path)
	db := engine.DbStore{
		Dict: dict,
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
