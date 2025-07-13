package engine

import (
	"sync"
	"time"
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
	Data       string
	Expiration time.Time
}
type RedisList struct {
	data []string
}

func (r RedisList) Type() string {
	return "LIST"
}
func (r RedisString) Type() string {
	return "STRING"
}
func (r RedisString) Value() interface{} {
	expiration := r.Expiration
	if expiration.Compare(time.Now()) >= 0 {
		return r.Data
	}
	return nil
}
