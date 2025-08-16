package tests

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Edge Cases and Error Handling Tests

// Protocol Error Handling Tests
func TestRedis_ProtocolErrorHandling(t *testing.T) {
	db := setupTestDB()

	errorTests := []struct {
		name        string
		command     string
		expectError bool
		description string
	}{
		{
			name:        "malformed array length",
			command:     "*abc\r\n$4\r\nPING\r\n",
			expectError: true,
			description: "array length is not a number",
		},
		{
			name:        "negative array length",
			command:     "*-1\r\n$4\r\nPING\r\n",
			expectError: true,
			description: "negative array length should be invalid",
		},
		{
			name:        "huge array length",
			command:     "*999999999\r\n$4\r\nPING\r\n",
			expectError: true,
			description: "unreasonably large array length",
		},
		{
			name:        "malformed bulk string length",
			command:     "*1\r\n$abc\r\nPING\r\n",
			expectError: true,
			description: "bulk string length is not a number",
		},
		{
			name:        "negative bulk string length",
			command:     "*1\r\n$-5\r\nPING\r\n",
			expectError: true,
			description: "negative bulk string length should be invalid",
		},
		{
			name:        "huge bulk string length",
			command:     "*1\r\n$999999999\r\nPING\r\n",
			expectError: true,
			description: "unreasonably large bulk string length",
		},
		{
			name:        "incomplete command",
			command:     "*2\r\n$3\r\nGET\r\n",
			expectError: true,
			description: "array claims 2 elements but only has 1",
		},
		{
			name:        "missing CRLF after array",
			command:     "*1$4\r\nPING\r\n",
			expectError: true,
			description: "missing CRLF after array length",
		},
		{
			name:        "missing CRLF after bulk length",
			command:     "*1\r\n$4PING\r\n",
			expectError: true,
			description: "missing CRLF after bulk string length",
		},
		{
			name:        "wrong bulk string length",
			command:     "*1\r\n$10\r\nPING\r\n",
			expectError: true,
			description: "bulk string length doesn't match actual string",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s but got result: %q", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}
		})
	}
}

// Command Validation Tests
func TestRedis_CommandValidation(t *testing.T) {
	db := setupTestDB()

	validationTests := []struct {
		name        string
		command     string
		expectError bool
		description string
	}{
		{
			name:        "unknown command",
			command:     "*1\r\n$7\r\nUNKNOWN\r\n",
			expectError: true,
			description: "unknown command should return error",
		},
		{
			name:        "SET with wrong arg count",
			command:     "*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n",
			expectError: true,
			description: "SET requires at least 2 arguments",
		},
		{
			name:        "SET with too many basic args",
			command:     "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n$5\r\nextra\r\n$4\r\nargs\r\n",
			expectError: true,
			description: "SET with too many arguments",
		},
		{
			name:        "GET with no args",
			command:     "*1\r\n$3\r\nGET\r\n",
			expectError: true,
			description: "GET requires 1 argument",
		},
		{
			name:        "GET with too many args",
			command:     "*3\r\n$3\r\nGET\r\n$3\r\nkey\r\n$5\r\nextra\r\n",
			expectError: true,
			description: "GET takes only 1 argument",
		},
		{
			name:        "KEYS with no args",
			command:     "*1\r\n$4\r\nKEYS\r\n",
			expectError: true,
			description: "KEYS requires 1 argument",
		},
		{
			name:        "PING with too many args",
			command:     "*3\r\n$4\r\nPING\r\n$5\r\nextra\r\n$4\r\nargs\r\n",
			expectError: true,
			description: "PING takes at most 1 argument",
		},
		{
			name:        "ECHO with no args",
			command:     "*1\r\n$4\r\nECHO\r\n",
			expectError: true,
			description: "ECHO requires 1 argument",
		},
		{
			name:        "ECHO with too many args",
			command:     "*3\r\n$4\r\nECHO\r\n$5\r\nhello\r\n$5\r\nworld\r\n",
			expectError: true,
			description: "ECHO takes only 1 argument",
		},
		{
			name:        "empty command array",
			command:     "*0\r\n",
			expectError: true,
			description: "empty command array should be invalid",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)

			if tt.expectError {
				if err == nil && !strings.Contains(strings.ToLower(result), "error") && !strings.HasPrefix(result, "-") {
					t.Errorf("Expected error for %s but got result: %q", tt.description, result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
				}
			}
		})
	}
}

// Special Character and Encoding Tests
func TestRedis_SpecialCharactersExtended(t *testing.T) {
	db := setupTestDB()

	specialTests := []struct {
		name  string
		key   string
		value string
	}{
		{"unicode key", "键", "value"},
		{"unicode value", "key", "值"},
		{"unicode both", "键", "值"},
		{"emoji key", "🔑", "value"},
		{"emoji value", "key", "🎉"},
		{"emoji both", "🔑", "🎉"},
		{"newlines in value", "key", "line1\nline2\nline3"},
		{"tabs in value", "key", "col1\tcol2\tcol3"},
		{"carriage returns", "key", "line1\rline2"},
		{"mixed whitespace", "key", " \t\n\r mixed \t\n\r "},
		{"special symbols", "key", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"null bytes", "key", "before\x00after"},
		{"high ascii", "key", "\x80\x90\xA0\xB0\xC0\xD0\xE0\xF0"},
		{"backslashes", "key", "path\\to\\file"},
		{"quotes", "key", `"quoted" and 'single'`},
		{"json-like", "key", `{"name":"value","num":123}`},
		{"xml-like", "key", `<tag attr="value">content</tag>`},
	}

	for _, tt := range specialTests {
		t.Run(tt.name, func(t *testing.T) {
			// SET
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(tt.key), tt.key, len(tt.value), tt.value)
			result, err := executeCommand(db, setCmd)
			if err != nil {
				t.Fatalf("SET failed for %s: %v", tt.name, err)
			}
			if result != "+OK\r\n" {
				t.Errorf("SET failed for %s: expected +OK, got %q", tt.name, result)
			}

			// GET
			getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(tt.key), tt.key)
			result, err = executeCommand(db, getCmd)
			if err != nil {
				t.Fatalf("GET failed for %s: %v", tt.name, err)
			}

			expected := fmt.Sprintf("+%s\r\n", tt.value)
			if result != expected {
				t.Errorf("GET failed for %s: expected %q, got %q", tt.name, expected, result)
			}
		})
	}
}

// Binary Data Tests
func TestRedis_BinaryData(t *testing.T) {
	db := setupTestDB()

	binaryTests := []struct {
		name  string
		key   string
		value []byte
	}{
		{"zero bytes", "binary1", []byte{0, 0, 0, 0}},
		{"full byte range", "binary2", []byte{0, 1, 127, 128, 255}},
		{"random bytes", "binary3", []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}},
		{"mixed ascii binary", "binary4", []byte("hello\x00world\x01\x02\x03")},
	}

	for _, tt := range binaryTests {
		t.Run(tt.name, func(t *testing.T) {
			value := string(tt.value)

			// SET
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(tt.key), tt.key, len(value), value)
			result, err := executeCommand(db, setCmd)
			if err != nil {
				t.Fatalf("SET failed for %s: %v", tt.name, err)
			}
			if result != "+OK\r\n" {
				t.Errorf("SET failed for %s: expected +OK, got %q", tt.name, result)
			}

			// GET
			getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(tt.key), tt.key)
			result, err = executeCommand(db, getCmd)
			if err != nil {
				t.Fatalf("GET failed for %s: %v", tt.name, err)
			}

			expected := fmt.Sprintf("+%s\r\n", value)
			if result != expected {
				t.Errorf("GET failed for %s: expected %q, got %q", tt.name, expected, result)
				t.Errorf("Expected bytes: %v", []byte(expected))
				t.Errorf("Actual bytes: %v", []byte(result))
			}
		})
	}
}

// Memory Efficiency Tests
func TestRedis_MemoryEfficiency(t *testing.T) {
	db := setupTestDB()

	// Test key/value size relationships
	sizeTests := []struct {
		name      string
		keySize   int
		valueSize int
		count     int
	}{
		{"small keys big values", 5, 1000, 100},
		{"big keys small values", 1000, 5, 100},
		{"equal sizes", 100, 100, 100},
		{"many tiny", 1, 1, 1000},
	}

	for _, tt := range sizeTests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()

			// Create test data
			for i := 0; i < tt.count; i++ {
				keyPrefix := strings.Repeat("k", max(1, tt.keySize-10))
				key := fmt.Sprintf("%s_%d", keyPrefix, i)
				if len(key) > tt.keySize {
					key = key[:tt.keySize] // Ensure exact size
				} else if len(key) < tt.keySize {
					key = key + strings.Repeat("k", tt.keySize-len(key))
				}

				value := strings.Repeat("v", max(1, tt.valueSize))

				setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
				result, err := executeCommand(db, setCmd)
				if err != nil {
					t.Fatalf("Failed to set key %d: %v", i, err)
				}
				if result != "+OK\r\n" {
					t.Fatalf("Unexpected SET result for key %d: %s", i, result)
				}
			}

			setTime := time.Since(startTime)

			// Test random access
			accessStart := time.Now()
			for i := 0; i < min(tt.count, 100); i++ {
				keyPrefix := strings.Repeat("k", max(1, tt.keySize-10))
				key := fmt.Sprintf("%s_%d", keyPrefix, i)
				if len(key) > tt.keySize {
					key = key[:tt.keySize]
				} else if len(key) < tt.keySize {
					key = key + strings.Repeat("k", tt.keySize-len(key))
				}

				getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
				result, err := executeCommand(db, getCmd)
				if err != nil {
					t.Errorf("Failed to get key %d: %v", i, err)
				}
				if !strings.HasPrefix(result, "+") {
					t.Errorf("Unexpected GET result for key %d: %s", i, result)
				}
			}
			accessTime := time.Since(accessStart)

			t.Logf("%s: SET %d items in %v, Access test in %v", tt.name, tt.count, setTime, accessTime)
		})
	}
}

// Timing and Race Condition Tests
func TestRedis_TimingRaceConditions(t *testing.T) {
	db := setupTestDB()

	// Test rapid SET/GET on same key
	key := "race_test_key"
	iterations := 1000

	for i := 0; i < iterations; i++ {
		value := fmt.Sprintf("value_%d", i)

		// SET
		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		result, err := executeCommand(db, setCmd)
		if err != nil {
			t.Fatalf("SET failed on iteration %d: %v", i, err)
		}
		if result != "+OK\r\n" {
			t.Fatalf("Unexpected SET result on iteration %d: %s", i, result)
		}

		// Immediate GET
		getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
		result, err = executeCommand(db, getCmd)
		if err != nil {
			t.Fatalf("GET failed on iteration %d: %v", i, err)
		}

		expected := fmt.Sprintf("+%s\r\n", value)
		if result != expected {
			t.Errorf("Race condition detected on iteration %d: expected %q, got %q", i, expected, result)
		}
	}
}

// Extreme Value Tests
func TestRedis_ExtremeValues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping extreme value tests in short mode")
	}

	db := setupTestDB()

	extremeTests := []struct {
		name       string
		key        string
		value      string
		shouldWork bool
	}{
		{
			name:       "very long key",
			key:        strings.Repeat("k", 10000),
			value:      "value",
			shouldWork: true,
		},
		{
			name:       "very long value",
			key:        "key",
			value:      strings.Repeat("v", 100000),
			shouldWork: true,
		},
		{
			name:       "extremely long value",
			key:        "big",
			value:      strings.Repeat("x", 1000000), // 1MB
			shouldWork: true,
		},
	}

	for _, tt := range extremeTests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()

			// SET
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(tt.key), tt.key, len(tt.value), tt.value)
			result, err := executeCommand(db, setCmd)

			setTime := time.Since(startTime)

			if tt.shouldWork {
				if err != nil {
					t.Fatalf("SET failed for %s: %v", tt.name, err)
				}
				if result != "+OK\r\n" {
					t.Errorf("SET failed for %s: expected +OK, got %q", tt.name, result)
				}

				// GET
				getStart := time.Now()
				getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(tt.key), tt.key)
				result, err = executeCommand(db, getCmd)
				getTime := time.Since(getStart)

				if err != nil {
					t.Fatalf("GET failed for %s: %v", tt.name, err)
				}

				expected := fmt.Sprintf("+%s\r\n", tt.value)
				if result != expected {
					t.Errorf("GET failed for %s: value mismatch", tt.name)
				}

				t.Logf("%s: SET in %v, GET in %v", tt.name, setTime, getTime)
			} else {
				if err == nil {
					t.Errorf("Expected failure for %s but got result: %q", tt.name, result)
				}
			}
		})
	}
}

// Cleanup and State Consistency Tests
func TestRedis_StateConsistency(t *testing.T) {
	db := setupTestDB()

	// Set multiple keys
	keys := []string{"state1", "state2", "state3", "state4", "state5"}

	for i, key := range keys {
		value := fmt.Sprintf("value_%d", i)
		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		executeCommand(db, setCmd)
	}

	// Verify all keys exist
	keysCmd := "*2\r\n$4\r\nKEYS\r\n$6\r\nstate*\r\n"
	result, err := executeCommand(db, keysCmd)
	if err != nil {
		t.Fatalf("KEYS command failed: %v", err)
	}

	// Check that we get the expected number of keys
	lines := strings.Split(result, "\r\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "*") {
		countStr := lines[0][1:]
		count, err := strconv.Atoi(countStr)
		if err == nil && count != len(keys) {
			t.Errorf("Expected %d keys, got %d", len(keys), count)
		}
	}

	// Verify each key has correct value
	for i, key := range keys {
		expectedValue := fmt.Sprintf("value_%d", i)
		getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
		result, err := executeCommand(db, getCmd)
		if err != nil {
			t.Errorf("Failed to get key %s: %v", key, err)
			continue
		}

		expected := fmt.Sprintf("+%s\r\n", expectedValue)
		if result != expected {
			t.Errorf("Key %s: expected %q, got %q", key, expected, result)
		}
	}
}

// Helper function for min operation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function for max operation
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
