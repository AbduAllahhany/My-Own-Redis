package tests

import (
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/utils"
)

func TestGenerateID(t *testing.T) {
	// Test that GenerateID returns non-empty string
	id1 := utils.GenerateID()
	if id1 == "" {
		t.Error("GenerateID returned empty string")
	}

	// Test that multiple calls return different IDs
	id2 := utils.GenerateID()
	if id1 == id2 {
		t.Error("GenerateID returned same ID for consecutive calls")
	}

	// Test ID format/length (assuming it should be reasonable length)
	if len(id1) < 8 {
		t.Errorf("GenerateID returned very short ID: %s", id1)
	}

	// Test that ID contains only valid characters
	for _, char := range id1 {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			t.Errorf("GenerateID returned ID with invalid character: %c in %s", char, id1)
		}
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	// Test uniqueness over many generations
	idSet := make(map[string]bool)
	numIDs := 1000

	for i := 0; i < numIDs; i++ {
		id := utils.GenerateID()
		if idSet[id] {
			t.Errorf("GenerateID returned duplicate ID: %s", id)
		}
		idSet[id] = true
	}

	if len(idSet) != numIDs {
		t.Errorf("Expected %d unique IDs, got %d", numIDs, len(idSet))
	}
}

func TestGenerateID_ThreadSafety(t *testing.T) {
	// Test that GenerateID is thread-safe
	numGoroutines := 10
	numIDsPerGoroutine := 100
	idChan := make(chan string, numGoroutines*numIDsPerGoroutine)

	// Start multiple goroutines generating IDs
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numIDsPerGoroutine; j++ {
				idChan <- utils.GenerateID()
			}
		}()
	}

	// Collect all IDs
	idSet := make(map[string]bool)
	totalExpected := numGoroutines * numIDsPerGoroutine

	for i := 0; i < totalExpected; i++ {
		id := <-idChan
		if idSet[id] {
			t.Errorf("Thread safety test: duplicate ID generated: %s", id)
		}
		idSet[id] = true
	}

	if len(idSet) != totalExpected {
		t.Errorf("Thread safety test: expected %d unique IDs, got %d", totalExpected, len(idSet))
	}
}

// Benchmark tests
func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.GenerateID()
	}
}

func BenchmarkGenerateID_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			utils.GenerateID()
		}
	})
}

// Test edge cases and assumptions
func TestGenerateID_Properties(t *testing.T) {
	id := utils.GenerateID()

	// Test that ID doesn't start or end with special characters
	if strings.HasPrefix(id, "-") || strings.HasSuffix(id, "-") {
		t.Errorf("ID should not start or end with dash: %s", id)
	}

	if strings.HasPrefix(id, "_") || strings.HasSuffix(id, "_") {
		t.Errorf("ID should not start or end with underscore: %s", id)
	}

	// Test that ID doesn't contain consecutive special characters
	if strings.Contains(id, "--") || strings.Contains(id, "__") {
		t.Errorf("ID should not contain consecutive special characters: %s", id)
	}
}

func TestGenerateID_Length(t *testing.T) {
	// Test that ID length is consistent and reasonable
	lengths := make(map[int]int)
	numTests := 100

	for i := 0; i < numTests; i++ {
		id := utils.GenerateID()
		lengths[len(id)]++
	}

	// Check that we don't have too much variance in length
	if len(lengths) > 3 {
		t.Errorf("Too much variance in ID lengths: %v", lengths)
	}

	// Check that all lengths are reasonable (between 8 and 64 characters)
	for length := range lengths {
		if length < 8 || length > 64 {
			t.Errorf("Unreasonable ID length: %d", length)
		}
	}
}
