package server

import (
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"strconv"
	"strings"
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

// string command
// optional args
// [NX | XX]
// [GET]
// [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | KEEPTTL]
func Set(request *Request) []byte {
	store := request.Serv.Db
	args := request.Args
	if len(args) < 2 || len(args) > 5 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	mu := store.Mu
	mu.Lock()
	dict := store.Dict
	key := args[0]
	value := args[1]

	//optional
	exIndex := IndexOf(args, "EX")
	pxIndex := IndexOf(args, "PX")
	var ttl time.Duration
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
	var ex time.Time
	if ttl != 0 {
		ex = time.Now().Add(ttl)
	}
	dict[key] = engine.RedisString{
		Data:       value,
		Expiration: ex,
	}
	mu.Unlock()

	if Contains(args, "GET") {
		req := Request{
			Serv: request.Serv,
			Args: []string{
				args[0],
			},
			Conn: request.Conn,
		}
		return Get(&req)
	}
	return resp.SimpleStringDecoder("OK")
}
func Get(request *Request) []byte {
	store := request.Serv.Db
	args := request.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error")
	}
	mu := store.Mu
	mu.RLock()
	defer mu.RUnlock()
	dict := store.Dict
	obj := dict[args[0]]
	if obj == nil {
		return []byte(resp.Nil)
	}
	if strings.ToUpper(obj.Type()) != "STRING" {
		return resp.ErrorDecoder("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	x := obj.Value()
	if x == nil {
		return []byte(resp.Nil)
	}
	str := x.(string)
	return resp.SimpleStringDecoder(str)
}
