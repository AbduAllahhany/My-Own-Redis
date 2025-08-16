package tests

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/server"
)

// Simple benchmark tests that don't require a running server

func BenchmarkRedisCore_SetGet(b *testing.B) {
	// Initialize commands
	server.InitCommands()

	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key"
		value := "benchmark_value"

		// SET operation
		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()

		// GET operation
		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value()
		}
		db.Mu.RUnlock()
	}
}

func BenchmarkRedisCore_SetGetWithExpiration(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "benchmark_key"
		value := "benchmark_value"
		expiration := time.Now().Add(time.Hour)

		// SET operation with expiration
		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: expiration,
		}
		db.Mu.Unlock()

		// GET operation
		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value()
		}
		db.Mu.RUnlock()
	}
}

func BenchmarkRedisCore_ConcurrentOperations(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "concurrent_key"
			value := "concurrent_value"

			if i%2 == 0 {
				// SET operation
				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			} else {
				// GET operation
				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}
			i++
		}
	})
}

func BenchmarkRedisCore_ManyKeys(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with many keys
	for i := 0; i < 10000; i++ {
		key := "preload_key_" + string(rune(i))
		value := "preload_value"
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "preload_key_" + string(rune(i%10000))

		// GET operation
		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value()
		}
		db.Mu.RUnlock()
	}
}

// BenchmarkStressCore_MassiveKeys tests performance with very large datasets
func BenchmarkStressCore_MassiveKeys(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with 100K keys
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("massive_key_%d", i)
		value := fmt.Sprintf("massive_value_%d", i)
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("massive_key_%d", i%100000)

		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value()
		}
		db.Mu.RUnlock()
	}
}

// BenchmarkStressCore_RapidUpdates tests rapid key updates
func BenchmarkStressCore_RapidUpdates(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	key := "rapid_update_key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := fmt.Sprintf("rapid_value_%d", i)

		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()
	}
}

// BenchmarkStressCore_MaxParallelism tests maximum parallelism stress
func BenchmarkStressCore_MaxParallelism(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Set GOMAXPROCS to maximum to test true parallelism
	originalProcs := runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	defer runtime.GOMAXPROCS(originalProcs)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		keyCounter := 0
		for pb.Next() {
			key := fmt.Sprintf("parallel_key_%d", keyCounter)
			value := fmt.Sprintf("parallel_value_%d", keyCounter)

			// Mix of operations
			if keyCounter%3 == 0 {
				// Write
				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			} else {
				// Read
				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}
			keyCounter++
		}
	})
}

// BenchmarkStressCore_MemoryChurn tests memory allocation/deallocation stress
func BenchmarkStressCore_MemoryChurn(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("churn_key_%d", i%1000) // Reuse keys to churn memory

		// Create large value to stress memory
		value := fmt.Sprintf("churn_value_%d_%s", i, string(make([]byte, 1024)))

		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()

		// Force GC every 100 operations
		if i%100 == 0 {
			runtime.GC()
		}
	}
}
