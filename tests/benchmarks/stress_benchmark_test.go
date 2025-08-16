package tests

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
)

// Stress tests for Redis implementation under high load conditions

// BenchmarkStress_MassiveParallelWrites tests high-volume concurrent write operations
func BenchmarkStress_MassiveParallelWrites(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	numWorkers := runtime.NumCPU() * 4 // Use more workers than CPU cores

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		workerID := rand.Intn(numWorkers)
		keyCounter := 0

		for pb.Next() {
			key := fmt.Sprintf("stress_worker_%d_key_%d", workerID, keyCounter)
			value := fmt.Sprintf("stress_value_%d_%d", workerID, keyCounter)

			db.Mu.Lock()
			(*db.Dict)[key] = engine.RedisString{
				Data:       value,
				Expiration: time.Time{},
			}
			db.Mu.Unlock()

			keyCounter++
		}
	})
}

// BenchmarkStress_ReadHeavyWorkload tests scenarios with 90% reads, 10% writes
func BenchmarkStress_ReadHeavyWorkload(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with data
	for i := 0; i < 100000; i++ {
		key := fmt.Sprintf("read_heavy_key_%d", i)
		value := fmt.Sprintf("read_heavy_value_%d", i)
		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: time.Time{},
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		operationCounter := 0

		for pb.Next() {
			// 90% reads, 10% writes
			if operationCounter%10 == 0 {
				// Write operation
				key := fmt.Sprintf("new_key_%d", rand.Intn(10000))
				value := fmt.Sprintf("new_value_%d", operationCounter)

				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			} else {
				// Read operation
				key := fmt.Sprintf("read_heavy_key_%d", rand.Intn(100000))

				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}

			operationCounter++
		}
	})
}

// BenchmarkStress_MemoryPressure tests behavior under memory pressure
func BenchmarkStress_MemoryPressure(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Large value size to create memory pressure
	largeValue := string(make([]byte, 10240)) // 10KB per value

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("memory_pressure_key_%d", i)

		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       largeValue,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()

		// Occasionally trigger GC to simulate memory pressure
		if i%1000 == 0 {
			runtime.GC()
		}

		// Remove old keys to prevent out of memory
		if i > 10000 {
			oldKey := fmt.Sprintf("memory_pressure_key_%d", i-10000)
			db.Mu.Lock()
			delete(*db.Dict, oldKey)
			db.Mu.Unlock()
		}
	}
}

// BenchmarkStress_ExpirationStorm tests massive expiration handling
func BenchmarkStress_ExpirationStorm(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with expiring keys
	baseTime := time.Now()
	for i := 0; i < 50000; i++ {
		key := fmt.Sprintf("expire_storm_key_%d", i)
		value := fmt.Sprintf("expire_storm_value_%d", i)
		expiration := baseTime.Add(time.Duration(i%1000) * time.Millisecond)

		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: expiration,
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		keyCounter := 0

		for pb.Next() {
			// Mix of operations on expiring keys
			key := fmt.Sprintf("expire_storm_key_%d", rand.Intn(50000))

			db.Mu.RLock()
			obj := (*db.Dict)[key]
			if obj != nil {
				_ = obj.Value() // This checks expiration
			}
			db.Mu.RUnlock()

			// Add new expiring key
			if keyCounter%10 == 0 {
				newKey := fmt.Sprintf("new_expire_key_%d", keyCounter)
				newValue := fmt.Sprintf("new_expire_value_%d", keyCounter)
				expiration := time.Now().Add(time.Duration(rand.Intn(2000)) * time.Millisecond)

				db.Mu.Lock()
				(*db.Dict)[newKey] = engine.RedisString{
					Data:       newValue,
					Expiration: expiration,
				}
				db.Mu.Unlock()
			}

			keyCounter++
		}
	})
}

// BenchmarkStress_ContentionLevel tests lock contention under high concurrency
func BenchmarkStress_ContentionLevel(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Small number of hot keys to create contention
	hotKeys := []string{"hot_key_1", "hot_key_2", "hot_key_3"}

	// Initialize hot keys
	for _, key := range hotKeys {
		(*db.Dict)[key] = engine.RedisString{
			Data:       "initial_value",
			Expiration: time.Time{},
		}
	}

	var writeCounter int64
	var readCounter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		operationCount := 0

		for pb.Next() {
			hotKey := hotKeys[rand.Intn(len(hotKeys))]

			// 30% writes on hot keys, 70% reads
			if operationCount%10 < 3 {
				// Write operation - high contention
				value := fmt.Sprintf("contention_value_%d", atomic.AddInt64(&writeCounter, 1))

				db.Mu.Lock()
				(*db.Dict)[hotKey] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()
			} else {
				// Read operation
				atomic.AddInt64(&readCounter, 1)

				db.Mu.RLock()
				obj := (*db.Dict)[hotKey]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}

			operationCount++
		}
	})
}

// BenchmarkStress_RapidKeyChurn tests rapid creation and deletion of keys
func BenchmarkStress_RapidKeyChurn(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	var keyCounter int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localKeys := make([]string, 0, 1000)

		for pb.Next() {
			keyID := atomic.AddInt64(&keyCounter, 1)
			key := fmt.Sprintf("churn_key_%d", keyID)
			value := fmt.Sprintf("churn_value_%d", keyID)

			// Create key
			db.Mu.Lock()
			(*db.Dict)[key] = engine.RedisString{
				Data:       value,
				Expiration: time.Time{},
			}
			db.Mu.Unlock()

			localKeys = append(localKeys, key)

			// Delete old keys to create churn
			if len(localKeys) > 100 {
				oldKey := localKeys[0]
				localKeys = localKeys[1:]

				db.Mu.Lock()
				delete(*db.Dict, oldKey)
				db.Mu.Unlock()
			}
		}
	})
}

// BenchmarkStress_MixedWorkload tests realistic mixed operations
func BenchmarkStress_MixedWorkload(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Pre-populate with some data
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("mixed_key_%d", i)
		value := fmt.Sprintf("mixed_value_%d", i)
		expiration := time.Time{}
		if i%10 == 0 {
			expiration = time.Now().Add(time.Duration(rand.Intn(5000)) * time.Millisecond)
		}

		(*db.Dict)[key] = engine.RedisString{
			Data:       value,
			Expiration: expiration,
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		operationCount := 0

		for pb.Next() {
			operation := rand.Intn(100)

			switch {
			case operation < 60: // 60% GET operations
				key := fmt.Sprintf("mixed_key_%d", rand.Intn(10000))
				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()

			case operation < 85: // 25% SET operations
				key := fmt.Sprintf("mixed_key_%d", rand.Intn(15000))
				value := fmt.Sprintf("updated_value_%d", operationCount)

				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()

			case operation < 95: // 10% SET with expiration
				key := fmt.Sprintf("expire_key_%d", rand.Intn(5000))
				value := fmt.Sprintf("expire_value_%d", operationCount)
				expiration := time.Now().Add(time.Duration(rand.Intn(10000)) * time.Millisecond)

				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: expiration,
				}
				db.Mu.Unlock()

			default: // 5% DELETE operations
				key := fmt.Sprintf("mixed_key_%d", rand.Intn(10000))
				db.Mu.Lock()
				delete(*db.Dict, key)
				db.Mu.Unlock()
			}

			operationCount++
		}
	})
}

// BenchmarkStress_LargeKeyOperations tests operations on large keys
func BenchmarkStress_LargeKeyOperations(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Generate large values (1MB each)
	largeValue := string(make([]byte, 1024*1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("large_key_%d", i%100) // Reuse keys to test overwrites

		db.Mu.Lock()
		(*db.Dict)[key] = engine.RedisString{
			Data:       largeValue,
			Expiration: time.Time{},
		}
		db.Mu.Unlock()

		// Read the large value
		db.Mu.RLock()
		obj := (*db.Dict)[key]
		if obj != nil {
			_ = obj.Value()
		}
		db.Mu.RUnlock()
	}
}

// BenchmarkStress_ThousandGoRoutines tests with very high goroutine count
func BenchmarkStress_ThousandGoRoutines(b *testing.B) {
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	numGoroutines := 1000
	operationsPerGoroutine := b.N / numGoroutines
	if operationsPerGoroutine < 1 {
		operationsPerGoroutine = 1
	}

	var wg sync.WaitGroup

	b.ResetTimer()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for op := 0; op < operationsPerGoroutine; op++ {
				key := fmt.Sprintf("thousand_key_%d_%d", goroutineID, op)
				value := fmt.Sprintf("thousand_value_%d_%d", goroutineID, op)

				// Write
				db.Mu.Lock()
				(*db.Dict)[key] = engine.RedisString{
					Data:       value,
					Expiration: time.Time{},
				}
				db.Mu.Unlock()

				// Read
				db.Mu.RLock()
				obj := (*db.Dict)[key]
				if obj != nil {
					_ = obj.Value()
				}
				db.Mu.RUnlock()
			}
		}(g)
	}

	wg.Wait()
}
