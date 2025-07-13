package rdb

import (
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/hdt3213/rdb/parser"
	"os"
	"time"
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
