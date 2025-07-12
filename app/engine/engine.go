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
	Data         string
	CreationTime time.Time
	TTL          time.Duration
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
	//	creation + ttl >= now
	t := r.CreationTime.Add(r.TTL)
	if t.Compare(time.Now()) >= 0 {
		return r.Data
	}
	return nil
}
