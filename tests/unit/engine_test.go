package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
)

func TestRedisString_Type(t *testing.T) {
	rs := engine.RedisString{Data: "test", Expiration: time.Time{}}
	if rs.Type() != "STRING" {
		t.Errorf("Expected type STRING, got %s", rs.Type())
	}
}

func TestRedisString_Value_NoExpiration(t *testing.T) {
	rs := engine.RedisString{Data: "test", Expiration: time.Time{}}
	value := rs.Value()
	if value != "test" {
		t.Errorf("Expected value 'test', got %v", value)
	}
}

func TestRedisString_Value_ValidExpiration(t *testing.T) {
	future := time.Now().Add(time.Hour)
	rs := engine.RedisString{Data: "test", Expiration: future}
	value := rs.Value()
	if value != "test" {
		t.Errorf("Expected value 'test', got %v", value)
	}
}

func TestRedisString_Value_ExpiredKey(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	rs := engine.RedisString{Data: "test", Expiration: past}
	value := rs.Value()
	if value != nil {
		t.Errorf("Expected nil for expired key, got %v", value)
	}
}

func TestRedisString_HasExpiration(t *testing.T) {
	tests := []struct {
		name       string
		expiration time.Time
		expected   bool
	}{
		{"No expiration", time.Time{}, false},
		{"With expiration", time.Now().Add(time.Hour), true},
		{"Past expiration", time.Now().Add(-time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := engine.RedisString{Data: "test", Expiration: tt.expiration}
			if rs.HasExpiration() != tt.expected {
				t.Errorf("Expected HasExpiration() to be %v, got %v", tt.expected, rs.HasExpiration())
			}
		})
	}
}

func TestRedisString_GetExpiration(t *testing.T) {
	expiration := time.Now().Add(time.Hour)
	rs := engine.RedisString{Data: "test", Expiration: expiration}
	if !rs.GetExpiration().Equal(expiration) {
		t.Errorf("Expected expiration %v, got %v", expiration, rs.GetExpiration())
	}
}

func TestRedisList_Type(t *testing.T) {
	rl := engine.RedisList{Data: []string{"item1", "item2"}, Expiration: time.Time{}}
	if rl.Type() != "LIST" {
		t.Errorf("Expected type LIST, got %s", rl.Type())
	}
}

func TestDbStore_ConcurrentAccess(t *testing.T) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Test concurrent reads and writes
	done := make(chan bool, 10)

	// Start 5 writers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				db.Mu.Lock()
				(*db.Dict)[string(rune('a'+id))] = engine.RedisString{
					Data:       "value",
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			}
			done <- true
		}(i)
	}

	// Start 5 readers
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				db.Mu.RLock()
				_ = (*db.Dict)[string(rune('a'+id))]
				db.Mu.RUnlock()
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDbStore_MemoryManagement(t *testing.T) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Add many keys
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       "value" + string(rune(i)),
			Expiration: time.Time{},
		}
		db.Mu.Unlock()
	}

	// Verify all keys exist
	db.Mu.RLock()
	count := len(*db.Dict)
	db.Mu.RUnlock()

	if count != 1000 {
		t.Errorf("Expected 1000 keys, got %d", count)
	}
}

// Benchmarks
func BenchmarkRedisString_Value_NoExpiration(b *testing.B) {
	rs := engine.RedisString{Data: "test", Expiration: time.Time{}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Value()
	}
}

func BenchmarkRedisString_Value_WithExpiration(b *testing.B) {
	rs := engine.RedisString{Data: "test", Expiration: time.Now().Add(time.Hour)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Value()
	}
}

func BenchmarkDbStore_WriteOperations(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       "value",
			Expiration: time.Time{},
		}
		db.Mu.Unlock()
	}
}

func BenchmarkDbStore_ReadOperations(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		key := "key" + string(rune(i))
		(*db.Dict)[key] = engine.RedisString{
			Data:       "value",
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "key" + string(rune(i%1000))
		db.Mu.RLock()
		_ = (*db.Dict)[key]
		db.Mu.RUnlock()
	}
}

func BenchmarkDbStore_ConcurrentReadWrite(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		key := "key" + string(rune(i))
		(*db.Dict)[key] = engine.RedisString{
			Data:       "value",
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key" + string(rune(i%100))
			if i%2 == 0 {
				// Read operation
				db.Mu.RLock()
				_ = (*db.Dict)[key]
				db.Mu.RUnlock()
			} else {
				// Write operation
				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       "value",
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			}
			i++
		}
	})
}
