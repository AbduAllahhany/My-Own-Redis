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

// string command
// optional args
// [NX | XX]
// [GET]
// [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | KEEPTTL]
func set(request *Request) ([]byte, error) {
	args := request.Cmd.Args
	if len(args) < 2 {
		return resp.ErrorDecoder("ERR wrong number of arguments for 'set' command"), ErrInvalidFormat
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
				return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
			}
			seconds, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorDecoder("ERR invalid EX time"), ErrInvalidFormat
			}
			ttl = time.Second * time.Duration(seconds)
			i++
		case "PX":
			if i+1 >= len(args) {
				return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
			}
			ms, err := strconv.Atoi(args[i+1])
			if err != nil {
				return resp.ErrorDecoder("ERR invalid PX time"), ErrInvalidFormat
			}
			ttl = time.Millisecond * time.Duration(ms)
			i++
		case "GET":
			returnOldValue = true
		default:
			return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
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
		return resp.SimpleStringDecoder(oldValue), nil
	}

	return resp.SimpleStringDecoder("OK"), nil
}
func get(request *Request) ([]byte, error) {
	store := request.Serv.Db
	args := request.Cmd.Args
	if len(args) != 1 {
		return resp.ErrorDecoder("ERR syntax error"), ErrInvalidFormat
	}
	mu := store.Mu
	mu.RLock()
	defer mu.RUnlock()
	dict := store.Dict
	obj := (*dict)[args[0]]
	if obj == nil {
		return []byte(resp.Nil), nil
	}
	if strings.ToUpper(obj.Type()) != "STRING" {
		return resp.ErrorDecoder("WRONGTYPE Operation against a key holding the wrong kind of value"), ErrInvalidFormat
	}
	x := obj.Value()
	if x == nil {
		return []byte(resp.Nil), nil
	}
	str := x.(string)
	if len(str) == 0 {
		return resp.BulkStringDecoder(str), nil
	}
	return resp.SimpleStringDecoder(str), nil

}
