package rdb

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/hdt3213/rdb/encoder"
	"github.com/hdt3213/rdb/parser"
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

func GenerateRDBBinary(dict *map[string]engine.RedisObj) ([]byte, error) {
	var buf bytes.Buffer
	enc := encoder.NewEncoder(&buf)
	err := enc.WriteHeader()
	if err != nil {
		return nil, err
	}

	// redis-ver = "7.2.0"
	if err := enc.WriteAux("redis-ver", "7.2.0"); err != nil {
		return nil, fmt.Errorf("failed to write redis-ver: %w", err)
	}

	// redis-bits = 64 (encoded as c040)
	if err := enc.WriteAux("redis-bits", "64"); err != nil {
		return nil, fmt.Errorf("failed to write redis-bits: %w", err)
	}

	// ctime = 1703845997 (encoded as c26d08bc65)
	if err := enc.WriteAux("ctime", "1703845997"); err != nil {
		return nil, fmt.Errorf("failed to write ctime: %w", err)
	}

	if err := enc.WriteAux("used-mem", "2960000000"); err != nil {
		return nil, fmt.Errorf("failed to write used-mem: %w", err)
	}

	if err := enc.WriteAux("aof-base", "0"); err != nil {
		return nil, fmt.Errorf("failed to write aof-base: %w", err)
	}

	// Only write DB header and objects if dict is not empty
	if len(*dict) > 0 {
		dictSize := len(*dict)
		expiringKeys := 0
		for _, redisObj := range *dict {
			if redisObj.HasExpiration() {
				expiringKeys++
			}
		}

		if err := enc.WriteDBHeader(0, uint64(dictSize), uint64(expiringKeys)); err != nil {
			return nil, fmt.Errorf("failed to write DB header: %w", err)
		}

		for key, redisObj := range *dict {
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
	}

	// Write end marker - this produces ff followed by checksum
	if err := enc.WriteEnd(); err != nil {
		return nil, fmt.Errorf("failed to write RDB end marker: %w", err)
	}
	return buf.Bytes(), nil
}
func GenerateRDBFile(dict *map[string]engine.RedisObj, filename string) error {
	data, err := GenerateRDBBinary(dict)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
