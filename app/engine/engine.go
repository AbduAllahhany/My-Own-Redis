package engine

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"strings"
	"sync"
)

type DbStore struct {
	Dict map[string]RedisObj
	Mu   sync.RWMutex
}

type RedisObj interface {
	Type() string
	Value() interface{}
}

type RedisString struct {
	data string
}

func (r RedisString) Type() string {
	return "STRING"
}
func (r RedisString) Value() interface{} {
	return r.data
}

type RedisList struct {
	data []string
}

func (r RedisList) Type() string {
	return "LIST"
}

// string command
func Set(store *DbStore, args []string) []byte {
	if len(args) != 2 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	mu := store.Mu
	dict := store.Dict
	mu.Lock()
	defer mu.Unlock()
	dict[args[0]] = RedisString{args[1]}
	return resp.SimpleStringDecoder("OK")
}
func Get(store *DbStore, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	dict := store.Dict
	obj := dict[args[0]]
	if obj == nil {
		return []byte(resp.Null)
	}
	if strings.ToUpper(obj.Type()) != "string" {
		return resp.ErrorDecoder("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	x := obj.Value()
	str := x.(string)
	return resp.SimpleStringDecoder(str)
}
