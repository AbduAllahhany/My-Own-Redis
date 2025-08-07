package server

import (
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
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
	args := request.Args
	if len(args) < 2 {
		return resp.ErrorDecoder("ERR wrong number of arguments for 'set' command")
	}

	key := args[0]
	value := args[1]

	var ttl time.Duration
	var expiration time.Time
	var returnOldValue bool

	// Parse optional flags
	for i := 2; i < len(args); i++ {
		arg := strings.ToUpper(args[i])
		switch arg {
		case "EX":
			if i+1 >= len(args) {
				return resp.ErrorDecoder("ERR syntax error")
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorDecoder("ERR invalid EX time")
			}
			ttl = time.Second * time.Duration(seconds)
			i++
		case "PX":
			if i+1 >= len(args) {
				return resp.ErrorDecoder("ERR syntax error")
			}
			ms, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorDecoder("ERR invalid PX time")
			}
			ttl = time.Millisecond * time.Duration(ms)
			i++
		case "GET":
			returnOldValue = true
		default:
			return resp.ErrorDecoder("ERR syntax error")
		}
	}

	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	store := request.Serv.Db
	store.Mu.Lock()
	defer store.Mu.Unlock()

	// Retrieve old value if requested
	var oldValue string
	if returnOldValue {
		if val, ok := (*store.Dict)[key]; ok {
			// Check if expired
			oldValue = val.Value().(string)
		}
	}

	// Set the new value
	(*store.Dict)[key] = engine.RedisString{
		Data:       value,
		Expiration: expiration,
	}

	if returnOldValue {
		return resp.SimpleStringDecoder(oldValue)
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
	obj := (*dict)[args[0]]
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
