package rdb

import (
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/hdt3213/rdb/parser"
	"os"
)

func Encode(path string) map[string]engine.RedisObj {
	rdbFile, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		_ = rdbFile.Close()
	}()
	dict := make(map[string]engine.RedisObj)
	decoder := parser.NewDecoder(rdbFile)
	err = decoder.Parse(func(o parser.RedisObject) bool {
		switch o.GetType() {
		case parser.StringType:
			str := o.(*parser.StringObject)
			dict[str.Key] = engine.RedisString{
				Data:       string(str.Value[0]),
				Expiration: *str.Expiration,
			}
		}
		return true
	})
	if err != nil {
		fmt.Println(err)
	}
	return dict
}
