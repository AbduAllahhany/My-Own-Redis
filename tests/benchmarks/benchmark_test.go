package tests

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/server"
)

// Comprehensive benchmark suite for Redis-like behavior

// BenchmarkRedis_CommandParsing tests command parsing performance
func BenchmarkRedis_CommandParsing(b *testing.B) {
	// Initialize commands
	server.InitCommands()

	testCommands := []string{
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
		"*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
		"*1\r\n$4\r\nPING\r\n",
		"*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n",
		"*5\r\n$3\r\nSET\r\n$6\r\ntestkey\r\n$9\r\ntestvalue\r\n$2\r\nEX\r\n$2\r\n10\r\n",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmdStr := testCommands[i%len(testCommands)]
		reader := bufio.NewReader(strings.NewReader(cmdStr))
		_, _ = server.ReadCommand(reader)
	}
}

// BenchmarkRedis_MemoryUsage tests memory usage patterns
func BenchmarkRedis_MemoryUsage(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("memory_key%d", i%10000)
		value := fmt.Sprintf("memory_value%d", i)

		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()

		// Occasionally read
		if i%10 == 0 {
			db.Mu.RLock()
			_ = (*db.Dict)[key]
			db.Mu.RUnlock()
		}
	}
}

// BenchmarkRedis_ExpirationHandling tests expiration handling performance
func BenchmarkRedis_ExpirationHandling(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with keys that have various expiration times
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("exp_test_key%d", i)
		expiration := time.Now().Add(time.Duration(i%100) * time.Millisecond)

		(*db.Dict)[key] = engine.RedisString{
			Data:       "value",
			Expiration: expiration,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("exp_test_key%d", i%1000)

		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value() // This checks expiration
		}
		db.Mu.RUnlock()
	}
}

// BenchmarkRedis_HighConcurrency tests high concurrency stress
func BenchmarkRedis_HighConcurrency(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with some data
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("concurrent_key_%d", i)
		value := fmt.Sprintf("concurrent_value_%d", i)
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		operationCount := 0
		for pb.Next() {
			keyIndex := operationCount % 10000
			key := fmt.Sprintf("concurrent_key_%d", keyIndex)

			// 70% reads, 30% writes for realistic stress
			if operationCount%10 < 7 {
				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			} else {
				value := fmt.Sprintf("updated_value_%d", operationCount)
				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			}
			operationCount++
		}
	})
}

// BenchmarkRedis_LockContention tests severe lock contention scenarios
func BenchmarkRedis_LockContention(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Single hot key to maximize contention
	hotKey := "super_hot_key"
	(*db.Dict)[hotKey] = engine.RedisString{
		Data:       "initial_value",
		Expiration: time.Time{},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		writeCount := 0
		for pb.Next() {
			// Alternate between reads and writes on the same key
			if writeCount%4 == 0 { // 25% writes
				value := fmt.Sprintf("contested_value_%d", writeCount)
				db.Mu.Lock()
				(*db.Dict)[hotKey] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			} else { // 75% reads
				db.Mu.RLock()
				obj := (*db.Dict)[hotKey]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}
			writeCount++
		}
	})
}
