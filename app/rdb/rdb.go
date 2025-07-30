package rdb

import (
	"bytes"
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/hdt3213/rdb/encoder"
	"github.com/hdt3213/rdb/parser"
	"os"
	"time"
)

func Encode(path string) map[string]engine.RedisObj {
	rdbFile, err := os.Open(path)
	dict := make(map[string]engine.RedisObj)
	if err != nil {
		return dict
	}
	defer func() {
		_ = rdbFile.Close()
	}()
	decoder := parser.NewDecoder(rdbFile)
	err = decoder.Parse(func(o parser.RedisObject) bool {
		switch o.GetType() {
		case parser.StringType:
			str := o.(*parser.StringObject)
			var exp time.Time
			if str.Expiration != nil {
				exp = *str.Expiration
			}
			dict[str.Key] = engine.RedisString{
				Data:       string(str.Value),
				Expiration: exp,
			}
		}
		return true
	})
	if err != nil {
		fmt.Println(err)
	}
	return dict
}

func GenerateRDBBinary(dict map[string]engine.RedisObj) ([]byte, error) {
	var buf bytes.Buffer
	enc := encoder.NewEncoder(&buf)
	if err := enc.WriteHeader(); err != nil {
		return nil, fmt.Errorf("failed to write RDB header: %w", err)
	}

	auxMap := map[string]string{
		"redis-ver":    "1.0.0",
		"redis-bits":   "64",
		"aof-preamble": "0",
	}
	for k, v := range auxMap {
		if err := enc.WriteAux(k, v); err != nil {
			return nil, fmt.Errorf("failed to write aux field %s: %w", k, err)
		}
	}

	// Calculate the number of keys and expiring keys in dict for the DB header
	dictSize := len(dict)
	expiringKeys := 0
	for _, redisObj := range dict {
		if redisObj.HasExpiration() {
			expiringKeys++
		}
	}

	if err := enc.WriteDBHeader(0, uint64(dictSize), uint64(expiringKeys)); err != nil {
		return nil, fmt.Errorf("failed to write DB header: %w", err)
	}

	for key, redisObj := range dict {
		var err error

		switch redisObj.Type() {
		case "STRING":
			if redisObj.HasExpiration() {
				expirationMs := uint64(redisObj.GetExpiration().Unix() * 1000)
				err = enc.WriteStringObject(key, []byte(redisObj.Value().(string)), encoder.WithTTL(expirationMs))
			} else {
				err = enc.WriteStringObject(key, []byte(redisObj.Value().(string)))
			}
		default:
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("failed to write object %s (type: %s): %w", key, redisObj.Type(), err)
		}
	}

	return buf.Bytes(), nil
}

func GenerateRDBFile(dict map[string]engine.RedisObj, filename string) error {
	data, err := GenerateRDBBinary(dict)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
