package tests

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func ParseRESPArray(respString string) []string {
	lines := strings.Split(strings.TrimSpace(respString), "\r\n")

	if len(lines) == 0 {
		return []string{}
	}

	// Check if it's a RESP array (starts with *)
	if !strings.HasPrefix(lines[0], "*") {
		// If not RESP format, return as single element
		return []string{respString}
	}

	// Parse array length
	arrayLengthStr := strings.TrimPrefix(lines[0], "*")
	arrayLength, err := strconv.Atoi(arrayLengthStr)
	if err != nil {
		return []string{respString}
	}

	if arrayLength == 0 {
		return []string{}
	}

	if arrayLength < 0 {
		// Null array
		return []string{}
	}

	result := make([]string, 0, arrayLength)
	i := 1

	for len(result) < arrayLength && i < len(lines) {
		if i >= len(lines) {
			break
		}

		line := lines[i]

		if strings.HasPrefix(line, "$") {
			// Bulk string
			lengthStr := strings.TrimPrefix(line, "$")
			length, err := strconv.Atoi(lengthStr)
			if err != nil {
				i++
				continue
			}

			if length < 0 {
				// Null bulk string
				result = append(result, "")
				i++
			} else if length == 0 {
				// Empty bulk string
				result = append(result, "")
				i += 2 // Skip the length line and empty content line
			} else {
				// Non-empty bulk string
				i++
				if i < len(lines) {
					result = append(result, lines[i])
					i++
				}
			}
		} else if strings.HasPrefix(line, "+") {
			// Simple string
			result = append(result, strings.TrimPrefix(line, "+"))
			i++
		} else if strings.HasPrefix(line, "-") {
			// Error
			result = append(result, strings.TrimPrefix(line, "-"))
			i++
		} else if strings.HasPrefix(line, ":") {
			// Integer
			result = append(result, strings.TrimPrefix(line, ":"))
			i++
		} else {
			// Unknown format, skip
			i++
		}
	}

	return result
}

// Advanced Redis functionality tests

// Protocol Compliance Tests
func TestRedis_ProtocolCompliance(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name        string
		command     string
		expected    string
		shouldError bool
	}{
		// Valid protocol formats
		{"valid array", "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n", "+OK\r\n", false},
		{"valid single command", "*1\r\n$4\r\nPING\r\n", "+PONG\r\n", false},

		// Invalid protocol formats should be handled gracefully
		{"empty command", "", "", true},
		{"invalid array start", "+3\r\n$3\r\nSET\r\n", "", true},
		{"missing CRLF", "*1\r\n$4\r\nPING", "", true},
		{"invalid bulk string length", "*2\r\n$999\r\nkey\r\n", "", true},
		{"negative array length", "*-2\r\n$3\r\nGET\r\n", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Key Pattern Matching Tests
func TestRedis_KeyPatternMatching(t *testing.T) {
	db := setupTestDB()

	// Setup test keys
	testKeys := []string{
		"user:1000:profile",
		"user:1000:settings",
		"user:1001:profile",
		"user:1001:settings",
		"session:abc123",
		"session:def456",
		"cache:page:home",
		"cache:page:about",
		"cache:api:users",
		"temp:data:123",
		"temp:data:456",
		"foo",
		"foobar",
		"foobaz",
		"bar",
		"baz",
	}

	for i, key := range testKeys {
		value := fmt.Sprintf("value_%d", i)
		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		executeCommand(db, setCmd)
	}

	patternTests := []struct {
		name           string
		pattern        string
		expectedCount  int
		mustContain    []string
		mustNotContain []string
	}{
		{
			name:          "wildcard all",
			pattern:       "*",
			expectedCount: len(testKeys),
			mustContain:   testKeys,
		},
		{
			name:           "prefix user",
			pattern:        "user:*",
			expectedCount:  4,
			mustContain:    []string{"user:1000:profile", "user:1000:settings", "user:1001:profile", "user:1001:settings"},
			mustNotContain: []string{"session:abc123", "cache:page:home"},
		},
		{
			name:           "specific user",
			pattern:        "user:1000:*",
			expectedCount:  2,
			mustContain:    []string{"user:1000:profile", "user:1000:settings"},
			mustNotContain: []string{"user:1001:profile", "session:abc123"},
		},
		{
			name:           "sessions",
			pattern:        "session:*",
			expectedCount:  2,
			mustContain:    []string{"session:abc123", "session:def456"},
			mustNotContain: []string{"user:1000:profile", "cache:page:home"},
		},
		{
			name:           "cache pages",
			pattern:        "cache:page:*",
			expectedCount:  2,
			mustContain:    []string{"cache:page:home", "cache:page:about"},
			mustNotContain: []string{"cache:api:users", "user:1000:profile"},
		},
		{
			name:          "temp data",
			pattern:       "temp:data:*",
			expectedCount: 2,
			mustContain:   []string{"temp:data:123", "temp:data:456"},
		},
		{
			name:           "foo prefix",
			pattern:        "foo*",
			expectedCount:  3,
			mustContain:    []string{"foo", "foobar", "foobaz"},
			mustNotContain: []string{"bar", "baz"},
		},
		{
			name:           "three letter words",
			pattern:        "???",
			expectedCount:  3,
			mustContain:    []string{"foo", "bar", "baz"},
			mustNotContain: []string{"foobar", "foobaz"},
		},
		{
			name:          "no matches",
			pattern:       "nonexistent:*",
			expectedCount: 0,
		},
	}

	for _, tt := range patternTests {
		t.Run(tt.name, func(t *testing.T) {
			keysCmd := fmt.Sprintf("*2\r\n$4\r\nKEYS\r\n$%d\r\n%s\r\n", len(tt.pattern), tt.pattern)
			result, err := executeCommand(db, keysCmd)
			returnedKeys := ParseRESPArray(result)
			if err != nil {
				t.Fatalf("KEYS command failed: %v", err)
			}

			// Parse array response
			lines := strings.Split(result, "\r\n")
			if len(lines) < 1 || !strings.HasPrefix(lines[0], "*") {
				t.Errorf("Invalid KEYS response format: %q", result)
				return
			}

			countStr := lines[0][1:]
			count, err := strconv.Atoi(countStr)
			if err != nil {
				t.Errorf("Invalid array count in KEYS response: %s", countStr)
				return
			}

			if count != tt.expectedCount {
				t.Errorf("Pattern %q: expected %d keys, got %d", tt.pattern, tt.expectedCount, count)
			}

			// Check required keys are present
			for _, mustKey := range tt.mustContain {
				if !slices.Contains(returnedKeys, mustKey) {
					t.Errorf("Pattern %q: expected key %q not found", tt.pattern, mustKey)
				}
			}

			// Check forbidden keys are absent
			for _, forbiddenKey := range tt.mustNotContain {
				if slices.Contains(returnedKeys, forbiddenKey) {
					t.Errorf("Pattern %q: unexpected key %q found", tt.pattern, forbiddenKey)
				}
			}
		})
	}
}

// Data Type Validation Tests
func TestRedis_DataTypeValidation(t *testing.T) {
	db := setupTestDB()

	// Set up different data types (all strings in this implementation)
	executeCommand(db, "*3\r\n$3\r\nSET\r\n$10\r\nstring_key\r\n$11\r\nstring_data\r\n")

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		// String operations on string keys should work
		{"get string", "*2\r\n$3\r\nGET\r\n$10\r\nstring_key\r\n", "+string_data\r\n"},
		{"set string", "*3\r\n$3\r\nSET\r\n$10\r\nstring_key\r\n$8\r\nnew_data\r\n", "+OK\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Boundary Value Tests
func TestRedis_BoundaryValues(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name        string
		key         string
		value       string
		description string
	}{
		{"single char key", "a", "value", "minimal key length"},
		{"single char value", "key", "a", "minimal value length"},
		{"long key", strings.Repeat("k", 100), "value", "100 character key"},
		{"long value", "key", strings.Repeat("v", 1000), "1000 character value"},
		{"empty value", "empty", "", "empty string value"},
		{"numeric key", "12345", "value", "numeric key"},
		{"numeric value", "key", "67890", "numeric value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// SET
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(tt.key), tt.key, len(tt.value), tt.value)
			result, err := executeCommand(db, setCmd)
			if err != nil {
				t.Fatalf("SET failed for %s: %v", tt.description, err)
			}
			if result != "+OK\r\n" {
				t.Errorf("SET failed for %s: expected +OK, got %q", tt.description, result)
			}

			// GET
			getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(tt.key), tt.key)
			result, err = executeCommand(db, getCmd)
			if err != nil {
				t.Fatalf("GET failed for %s: %v", tt.description, err)
			}
			expected := fmt.Sprintf("+%s\r\n", tt.value)
			// Handle empty value case - might return bulk string format
			if tt.value == "" && (result == "$0\r\n\r\n" || result == "+\r\n") {
				// Both formats are acceptable for empty values
			} else if result != expected {
				t.Errorf("GET failed for %s: expected %q, got %q", tt.description, expected, result)
			}
		})
	}
}

// High Load Stress Tests
func TestRedis_HighLoadStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	db := setupTestDB()

	// Test with high concurrency
	numGoroutines := 50
	numOperationsPerGoroutine := 100
	var wg sync.WaitGroup
	var errorCount int32
	var successCount int32

	wg.Add(numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				key := fmt.Sprintf("stress_key_%d_%d", goroutineID, j)
				value := fmt.Sprintf("stress_value_%d_%d", goroutineID, j)

				// SET operation
				setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
				result, err := executeCommand(db, setCmd)
				if err != nil || result != "+OK\r\n" {
					errorCount++
					continue
				}

				// GET operation
				getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
				result, err = executeCommand(db, getCmd)
				expected := fmt.Sprintf("+%s\r\n", value)
				if err != nil || result != expected {
					errorCount++
					continue
				}

				// KEYS operation (less frequent)
				if j%10 == 0 {
					pattern := fmt.Sprintf("stress_key_%d_*", goroutineID)
					keysCmd := fmt.Sprintf("*2\r\n$4\r\nKEYS\r\n$%d\r\n%s\r\n", len(pattern), pattern)
					_, err := executeCommand(db, keysCmd)
					if err != nil {
						errorCount++
						continue
					}
				}

				successCount++
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	totalOperations := numGoroutines * numOperationsPerGoroutine * 2 // SET + GET
	operationsPerSecond := float64(totalOperations) / duration.Seconds()

	t.Logf("Stress test completed in %v", duration)
	t.Logf("Total operations: %d", totalOperations)
	t.Logf("Operations per second: %.2f", operationsPerSecond)
	t.Logf("Success count: %d", successCount)
	t.Logf("Error count: %d", errorCount)

	if errorCount > 0 {
		t.Errorf("Stress test had %d errors", errorCount)
	}

	// Verify final state
	keysResult, _ := executeCommand(db, "*2\r\n$4\r\nKEYS\r\n$11\r\nstress_key_*\r\n")
	lines := strings.Split(keysResult, "\r\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "*") {
		countStr := lines[0][1:]
		count, err := strconv.Atoi(countStr)
		if err == nil {
			expectedKeys := numGoroutines * numOperationsPerGoroutine
			if count != expectedKeys {
				t.Errorf("Expected %d keys after stress test, got %d", expectedKeys, count)
			}
		}
	}
}

// Memory Pressure Tests
func TestRedis_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	db := setupTestDB()

	// Test with large number of keys
	numKeys := 10000
	keySize := 50
	valueSize := 200

	t.Logf("Creating %d keys with %d byte keys and %d byte values", numKeys, keySize, valueSize)

	startTime := time.Now()

	// Create many keys
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("mem_test_%s_%d", strings.Repeat("k", keySize-20), i)
		value := strings.Repeat("v", valueSize)

		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		result, err := executeCommand(db, setCmd)
		if err != nil {
			t.Fatalf("Failed to set key %d: %v", i, err)
		}
		if result != "+OK\r\n" {
			t.Fatalf("Unexpected result for key %d: %s", i, result)
		}

		// Progress indicator
		if i > 0 && i%1000 == 0 {
			t.Logf("Created %d/%d keys", i, numKeys)
		}
	}

	creationTime := time.Since(startTime)
	t.Logf("Key creation completed in %v", creationTime)

	// Test random access
	accessStartTime := time.Now()
	for i := 0; i < 1000; i++ {
		keyIndex := i % numKeys
		key := fmt.Sprintf("mem_test_%s_%d", strings.Repeat("k", keySize-20), keyIndex)

		getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
		result, err := executeCommand(db, getCmd)
		if err != nil {
			t.Errorf("Failed to get key %d: %v", keyIndex, err)
		}
		if !strings.HasPrefix(result, "+") {
			t.Errorf("Unexpected result for GET key %d: %s", keyIndex, result)
		}
	}

	accessTime := time.Since(accessStartTime)
	t.Logf("Random access test completed in %v", accessTime)

	// Test KEYS command performance with many keys
	keysStartTime := time.Now()

	keysResult, err := executeCommand(db, "*2\r\n$4\r\nKEYS\r\n$10\r\nmem_test_*\r\n")
	if err != nil {
		t.Fatalf("KEYS command failed: %v", err)
	}
	keysTime := time.Since(keysStartTime)
	t.Logf("KEYS command completed in %v", keysTime)

	// Verify count
	lines := strings.Split(keysResult, "\r\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "*") {
		countStr := lines[0][1:]
		count, err := strconv.Atoi(countStr)
		if err == nil && count != numKeys {
			t.Errorf("KEYS returned %d keys, expected %d", count, numKeys)
		}
	}
}

// Expiration Accuracy Tests
func TestRedis_ExpirationAccuracy(t *testing.T) {
	db := setupTestDB()

	// Test multiple keys with different expiration times
	expirationTests := []struct {
		key          string
		ttlMs        int
		shouldExpire bool
		waitMs       int
	}{
		{"expire_10ms", 10, true, 20},
		{"expire_50ms", 50, true, 70},
		{"expire_100ms", 100, true, 120},
		{"expire_500ms", 500, false, 100}, // Should not expire yet
		{"no_expire", 0, false, 100},      // No expiration set
	}

	// Set keys with different TTLs
	for _, test := range expirationTests {
		if test.ttlMs > 0 {
			setCmd := fmt.Sprintf("*5\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$%d\r\n%d\r\n",
				len(test.key), test.key, len(strconv.Itoa(test.ttlMs)), test.ttlMs)
			result, err := executeCommand(db, setCmd)
			if err != nil || result != "+OK\r\n" {
				t.Fatalf("Failed to set key %s with TTL: %v", test.key, err)
			}
		} else {
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$5\r\nvalue\r\n", len(test.key), test.key)
			executeCommand(db, setCmd)
		}
	}

	// Wait and check expiration
	maxWait := 0
	for _, test := range expirationTests {
		if test.waitMs > maxWait {
			maxWait = test.waitMs
		}
	}
	time.Sleep(time.Duration(maxWait) * time.Millisecond)

	// Verify expiration behavior
	for _, test := range expirationTests {
		getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(test.key), test.key)
		result, err := executeCommand(db, getCmd)
		if err != nil {
			t.Errorf("Failed to get key %s: %v", test.key, err)
			continue
		}

		if test.shouldExpire {
			if result != "$-1\r\n" {
				t.Errorf("Key %s should have expired but got: %q", test.key, result)
			}
		} else {
			if result == "$-1\r\n" {
				t.Errorf("Key %s should not have expired but got: %q", test.key, result)
			}
		}
	}
}

// Configuration and Info Tests
func TestRedis_ConfigAndInfo(t *testing.T) {
	db := setupTestDB()

	// Test INFO command (if implemented)
	infoCmd := "*2\r\n$4\r\nINFO\r\n$11\r\nreplication\r\n"
	result, err := executeCommand(db, infoCmd)

	// INFO might not be implemented, so we check if it returns an error or valid response
	if err == nil {
		if !strings.Contains(result, "role:") {
			t.Logf("INFO command returned: %q", result)
		}
	} else {
		t.Logf("INFO command not implemented or failed: %v", err)
	}

	// Test CONFIG command (if implemented)
	configCmd := "*3\r\n$6\r\nCONFIG\r\n$3\r\nGET\r\n$4\r\nport\r\n"
	result, err = executeCommand(db, configCmd)

	if err == nil {
		t.Logf("CONFIG GET port returned: %q", result)
	} else {
		t.Logf("CONFIG command not implemented or failed: %v", err)
	}
}
