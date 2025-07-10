package engine

import (
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var expirationOptions = map[string]time.Duration{
	"PX": time.Millisecond,
	"EX": time.Second,
}

func Contains(slice []string, item string) bool {
	return IndexOf(slice, item) != -1
}

func IndexOf(slice []string, item string) int {
	for i, v := range slice { // loop is still here, but abstracted away
		if strings.ToUpper(v) == strings.ToUpper(item) {
			return i
		}
	}
	return -1
}

type DbStore struct {
	Dict map[string]RedisObj
	Mu   sync.RWMutex
}

type RedisObj interface {
	Type() string
	Value() interface{}
}

type RedisString struct {
	data         string
	creationTime time.Time
	ttl          time.Duration
}

func (r RedisString) Type() string {
	return "STRING"
}
func (r RedisString) Value() interface{} {
	//	creation + ttl >= now
	t := r.creationTime.Add(r.ttl)
	if t.Compare(time.Now()) >= 0 {
		return r.data
	}
	return nil
}

type RedisList struct {
	data []string
}

func (r RedisList) Type() string {
	return "LIST"
}

// string command

// optional args
// [NX | XX]
// [GET]
// [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | KEEPTTL]
func Set(store *DbStore, args []string) []byte {
	if len(args) < 2 || len(args) > 5 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	mu := store.Mu
	mu.Lock()
	defer mu.Unlock()
	dict := store.Dict
	key := args[0]
	value := args[1]

	//optional
	exIndex := IndexOf(args, "EX")
	pxIndex := IndexOf(args, "PX")
	var ttl time.Duration = 1<<63 - 1
	if exIndex != -1 && pxIndex == -1 {
		amount, err := strconv.Atoi(args[exIndex+1])
		if err != nil {
			return resp.ErrorDecoder("ERR value is not an integer or out of range")
		}
		ttl = expirationOptions[strings.ToUpper(args[exIndex])] * time.Duration(amount)
	} else if pxIndex != -1 && exIndex == -1 {
		amount, err := strconv.Atoi(args[pxIndex+1])
		if err != nil {
			return resp.ErrorDecoder("ERR value is not an integer or out of range")
		}
		ttl = expirationOptions[strings.ToUpper(args[pxIndex])] * time.Duration(amount)
	}

	dict[key] = RedisString{
		data:         value,
		creationTime: time.Now(),
		ttl:          ttl,
	}

	if Contains(args, "GET") {
		return Get(store, []string{
			args[0],
		})
	}
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
	if strings.ToUpper(obj.Type()) != "STRING" {
		return resp.ErrorDecoder("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	x := obj.Value()
	if x == nil {
		return []byte(resp.Null)
	}
	str := x.(string)
	return resp.SimpleStringDecoder(str)
}

// server commands
func Echo(store *DbStore, args []string) []byte {
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.BulkStringDecoder(args[0])
}

func Ping(store *DbStore, args []string) []byte {
	if len(args) != 0 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	return resp.SimpleStringDecoder("PONG")
}
