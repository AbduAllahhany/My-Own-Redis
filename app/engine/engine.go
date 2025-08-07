package engine

import (
	"sync"
	"time"
)

type DbStore struct {
	Dict *map[string]RedisObj
	Mu   sync.RWMutex
}

type RedisObj interface {
	Type() string
	Value() interface{}
	HasExpiration() bool
	GetExpiration() time.Time
}
type RedisString struct {
	Data       string
	Expiration time.Time
}
type RedisList struct {
	Data       []string
	Expiration time.Time
}

func (r RedisList) Type() string {
	return "LIST"
}

func (r RedisString) Type() string {
	return "STRING"
}
func (r RedisString) Value() interface{} {
	expiration := r.Expiration
	if expiration.Compare(time.Now()) >= 0 || expiration.IsZero() {
		return r.Data
	}
	return nil
}
func (r RedisString) HasExpiration() bool {
	return !r.Expiration.IsZero()
}
func (r RedisString) GetExpiration() time.Time {
	return r.Expiration
}
