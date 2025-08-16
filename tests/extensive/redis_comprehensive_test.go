package tests

import (
	"bufio"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/server"
)

// Comprehensive Redis-like tests covering all major functionality

func setupTestDB() *engine.DbStore {
	server.InitCommands()
	dict := make(map[string]engine.RedisObj)
	return &engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}
}

func executeCommand(db *engine.DbStore, command string) (string, error) {
	reader := bufio.NewReader(strings.NewReader(command))

	cmd, err := server.ReadCommand(reader)
	if err != nil {
		return "", err
	}

	request := &server.Request{
		Serv: &server.Server{
			Db: db,
		},
		Cmd: &cmd,
	}

	result, err := server.ProcessCommand(request)
	if err != nil {
		return string(result), err
	}
	return string(result), nil
}

// Basic String Operations Tests
func TestRedis_StringOperations(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"SET basic", "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n", "+OK\r\n"},
		{"GET basic", "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n", "+value\r\n"},
		{"SET overwrite", "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$8\r\nnewvalue\r\n", "+OK\r\n"},
		{"GET after overwrite", "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n", "+newvalue\r\n"},
		{"GET non-existent", "*2\r\n$3\r\nGET\r\n$7\r\nmissing\r\n", "$-1\r\n"},
		{"SET empty value", "*3\r\n$3\r\nSET\r\n$5\r\nempty\r\n$0\r\n\r\n", "+OK\r\n"},
		{"GET empty value", "*2\r\n$3\r\nGET\r\n$5\r\nempty\r\n", "$0\r\n\r\n"},
		{"SET with spaces", "*3\r\n$3\r\nSET\r\n$9\r\nkey space\r\n$11\r\nvalue space\r\n", "+OK\r\n"},
		{"GET with spaces", "*2\r\n$3\r\nGET\r\n$9\r\nkey space\r\n", "+value space\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)
			if err != nil && !strings.Contains(tt.expected, "ERR") {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// SET with Expiration Tests
func TestRedis_SetWithExpiration(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"SET with EX", "*5\r\n$3\r\nSET\r\n$6\r\nexkey1\r\n$5\r\nvalue\r\n$2\r\nEX\r\n$1\r\n1\r\n", "+OK\r\n"},
		{"SET with PX", "*5\r\n$3\r\nSET\r\n$6\r\nexkey2\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$4\r\n1000\r\n", "+OK\r\n"},
		{"SET with PX short", "*5\r\n$3\r\nSET\r\n$6\r\nexkey3\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$2\r\n50\r\n", "+OK\r\n"},
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

	// Test immediate access to keys with expiration
	result, _ := executeCommand(db, "*2\r\n$3\r\nGET\r\n$6\r\nexkey1\r\n")
	if result != "+value\r\n" {
		t.Errorf("Key should be accessible immediately after SET with EX")
	}

	// Test expiration
	time.Sleep(60 * time.Millisecond)
	result, _ = executeCommand(db, "*2\r\n$3\r\nGET\r\n$6\r\nexkey3\r\n")
	if result != "$-1\r\n" {
		t.Errorf("Key should be expired after 50ms, got: %q", result)
	}
}

// ECHO Command Tests
func TestRedis_EchoCommand(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"ECHO simple", "*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n", "$5\r\nhello\r\n"},
		{"ECHO empty", "*2\r\n$4\r\nECHO\r\n$0\r\n\r\n", "$0\r\n\r\n"},
		{"ECHO with spaces", "*2\r\n$4\r\nECHO\r\n$11\r\nhello world\r\n", "$11\r\nhello world\r\n"},
		{"ECHO special chars", "*2\r\n$4\r\nECHO\r\n$13\r\nhello\r\nworld!\r\n", "$13\r\nhello\r\nworld!\r\n"},
		{"ECHO numbers", "*2\r\n$4\r\nECHO\r\n$6\r\n123456\r\n", "$6\r\n123456\r\n"},
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

// PING Command Tests
func TestRedis_PingCommand(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"PING simple", "*1\r\n$4\r\nPING\r\n", "+PONG\r\n"},
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

// KEYS Command Tests
func TestRedis_KeysCommand(t *testing.T) {
	db := setupTestDB()

	// Set up test data
	setupCommands := []string{
		"*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$6\r\nvalue1\r\n",
		"*3\r\n$3\r\nSET\r\n$4\r\nkey2\r\n$6\r\nvalue2\r\n",
		"*3\r\n$3\r\nSET\r\n$8\r\ntest_key\r\n$5\r\nvalue\r\n",
		"*3\r\n$3\r\nSET\r\n$7\r\nuser:id\r\n$4\r\n1234\r\n",
		"*3\r\n$3\r\nSET\r\n$9\r\nuser:name\r\n$4\r\njohn\r\n",
	}

	for _, cmd := range setupCommands {
		executeCommand(db, cmd)
	}

	tests := []struct {
		name             string
		command          string
		expectedCount    int
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name:          "KEYS *",
			command:       "*2\r\n$4\r\nKEYS\r\n$1\r\n*\r\n",
			expectedCount: 5,
			shouldContain: []string{"key1", "key2", "test_key", "user:id", "user:name"},
		},
		{
			name:             "KEYS key*",
			command:          "*2\r\n$4\r\nKEYS\r\n$4\r\nkey*\r\n",
			expectedCount:    2,
			shouldContain:    []string{"key1", "key2"},
			shouldNotContain: []string{"test_key", "user:id"},
		},
		{
			name:             "KEYS user:*",
			command:          "*2\r\n$4\r\nKEYS\r\n$6\r\nuser:*\r\n",
			expectedCount:    2,
			shouldContain:    []string{"user:id", "user:name"},
			shouldNotContain: []string{"key1", "test_key"},
		},
		{
			name:          "KEYS test_*",
			command:       "*2\r\n$4\r\nKEYS\r\n$6\r\ntest_*\r\n",
			expectedCount: 1,
			shouldContain: []string{"test_key"},
		},
		{
			name:          "KEYS nomatch*",
			command:       "*2\r\n$4\r\nKEYS\r\n$8\r\nnomatch*\r\n",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommand(db, tt.command)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Parse array response
			if !strings.HasPrefix(result, "*") {
				t.Errorf("Expected array response, got: %q", result)
				return
			}

			lines := strings.Split(result, "\r\n")
			if len(lines) < 1 {
				t.Errorf("Invalid response format: %q", result)
				return
			}

			countStr := lines[0][1:] // Remove the '*'
			count, err := strconv.Atoi(countStr)
			if err != nil {
				t.Errorf("Invalid array count: %s", countStr)
				return
			}

			if count != tt.expectedCount {
				t.Errorf("Expected %d keys, got %d", tt.expectedCount, count)
			}
			returnedKeys := ParseRESPArray(result)
			// Check if expected keys are present
			for _, expectedKey := range tt.shouldContain {
				if !slices.Contains(returnedKeys, expectedKey) {
					t.Errorf("Expected key %q not found in result: %q", expectedKey, result)
				}
			}

			// Check if unwanted keys are absent
			for _, unwantedKey := range tt.shouldNotContain {
				if slices.Contains(returnedKeys, unwantedKey) {
					t.Errorf("Unwanted key %q found in result: %q", unwantedKey, result)
				}
			}
		})
	}
}

// Error Handling Tests
func TestRedis_ErrorHandling(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"SET wrong args", "*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n", "-ERR wrong number of arguments for 'set' command\r\n"},
		{"GET wrong args", "*1\r\n$3\r\nGET\r\n", "-ERR syntax error\r\n"},
		{"GET too many args", "*3\r\n$3\r\nGET\r\n$3\r\nkey\r\n$5\r\nextra\r\n", "-ERR syntax error\r\n"},
		{"ECHO wrong args", "*1\r\n$4\r\nECHO\r\n", "-ERR syntax error\r\n"},
		{"PING wrong args", "*2\r\n$4\r\nPING\r\n$5\r\nextra\r\n", "-ERR syntax error\r\n"},
		{"Unknown command", "*1\r\n$7\r\nUNKNOWN\r\n", "-ERR unknown command\r\n"},
		{"SET invalid EX", "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n$2\r\nEX\r\n$3\r\nabc\r\n", "-ERR invalid EX time\r\n"},
		{"SET invalid PX", "*5\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$3\r\nxyz\r\n", "-ERR invalid PX time\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := executeCommand(db, tt.command)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Concurrent Access Tests
func TestRedis_ConcurrentAccess(t *testing.T) {
	db := setupTestDB()

	var wg sync.WaitGroup
	numGoroutines := 20
	numOperations := 50

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent_key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d", j)

				// SET operation
				setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
				result, err := executeCommand(db, setCmd)
				if err != nil {
					t.Errorf("SET error in goroutine %d: %v", id, err)
					return
				}
				if result != "+OK\r\n" {
					t.Errorf("SET failed in goroutine %d: %s", id, result)
					return
				}

				// GET operation
				getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)
				result, err = executeCommand(db, getCmd)
				if err != nil {
					t.Errorf("GET error in goroutine %d: %v", id, err)
					return
				}
				expected := fmt.Sprintf("+%s\r\n", value)
				if result != expected {
					t.Errorf("GET failed in goroutine %d: expected %q, got %q", id, expected, result)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify total number of keys
	keysResult, _ := executeCommand(db, "*2\r\n$4\r\nKEYS\r\n$1\r\n*\r\n")
	lines := strings.Split(keysResult, "\r\n")
	if len(lines) > 0 {
		countStr := lines[0][1:] // Remove the '*'
		count, err := strconv.Atoi(countStr)
		if err == nil {
			expectedCount := numGoroutines * numOperations
			if count != expectedCount {
				t.Errorf("Expected %d keys after concurrent operations, got %d", expectedCount, count)
			}
		}
	}
}

// Memory Usage and Cleanup Tests
func TestRedis_MemoryUsage(t *testing.T) {
	db := setupTestDB()

	// Test with many keys
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("memory_test_key_%d", i)
		value := fmt.Sprintf("memory_test_value_%d", i)
		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(value), value)
		result, err := executeCommand(db, setCmd)
		if err != nil {
			t.Fatalf("Failed to set key %d: %v", i, err)
		}
		if result != "+OK\r\n" {
			t.Fatalf("Unexpected result for key %d: %s", i, result)
		}
	}

	// Verify all keys exist
	keysResult, _ := executeCommand(db, "*2\r\n$4\r\nKEYS\r\n$1\r\n*\r\n")
	lines := strings.Split(keysResult, "\r\n")
	if len(lines) > 0 {
		countStr := lines[0][1:]
		count, err := strconv.Atoi(countStr)
		if err == nil && count != numKeys {
			t.Errorf("Expected %d keys, got %d", numKeys, count)
		}
	}

	// Test key overwriting
	for i := 0; i < numKeys/2; i++ {
		key := fmt.Sprintf("memory_test_key_%d", i)
		newValue := fmt.Sprintf("updated_value_%d", i)
		setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(key), key, len(newValue), newValue)
		executeCommand(db, setCmd)
	}

	// Verify key count remains the same
	keysResult, _ = executeCommand(db, "*2\r\n$4\r\nKEYS\r\n$1\r\n*\r\n")
	lines = strings.Split(keysResult, "\r\n")
	if len(lines) > 0 {
		countStr := lines[0][1:]
		count, err := strconv.Atoi(countStr)
		if err == nil && count != numKeys {
			t.Errorf("Key count changed after overwriting: expected %d, got %d", numKeys, count)
		}
	}
}

// Expiration Edge Cases Tests
func TestRedis_ExpirationEdgeCases(t *testing.T) {
	db := setupTestDB()

	// Test very short expiration
	result, _ := executeCommand(db, "*5\r\n$3\r\nSET\r\n$9\r\nshort_ttl\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$1\r\n1\r\n")
	if result != "+OK\r\n" {
		t.Errorf("Failed to set key with 1ms TTL")
	}

	// Key should expire very quickly
	time.Sleep(5 * time.Millisecond)
	result, _ = executeCommand(db, "*2\r\n$3\r\nGET\r\n$9\r\nshort_ttl\r\n")
	if result != "$-1\r\n" {
		t.Errorf("Key should have expired, got: %q", result)
	}

	// Test zero expiration (should not set expiration)
	executeCommand(db, "*5\r\n$3\r\nSET\r\n$6\r\nno_ttl\r\n$5\r\nvalue\r\n$2\r\nEX\r\n$1\r\n0\r\n")
	time.Sleep(10 * time.Millisecond)
	result, _ = executeCommand(db, "*2\r\n$3\r\nGET\r\n$6\r\nno_ttl\r\n")
	if result == "$-1\r\n" {
		t.Errorf("Key with 0 TTL should not expire")
	}

	// Test overwriting key with expiration
	executeCommand(db, "*5\r\n$3\r\nSET\r\n$12\r\noverwrite_me\r\n$6\r\nvalue1\r\n$2\r\nPX\r\n$4\r\n1000\r\n")
	executeCommand(db, "*3\r\n$3\r\nSET\r\n$12\r\noverwrite_me\r\n$6\r\nvalue2\r\n") // No expiration
	time.Sleep(50 * time.Millisecond)
	result, _ = executeCommand(db, "*2\r\n$3\r\nGET\r\n$12\r\noverwrite_me\r\n")
	if result != "+value2\r\n" {
		t.Errorf("Overwritten key should not expire, got: %q", result)
	}
}

// Case Sensitivity Tests
func TestRedis_CaseSensitivity(t *testing.T) {
	db := setupTestDB()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"lowercase set", "*3\r\n$3\r\nset\r\n$3\r\nkey\r\n$5\r\nvalue\r\n", "+OK\r\n"},
		{"uppercase GET", "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n", "+value\r\n"},
		{"mixed case ping", "*1\r\n$4\r\nPiNg\r\n", "+PONG\r\n"},
		{"lowercase echo", "*2\r\n$4\r\necho\r\n$5\r\nhello\r\n", "$5\r\nhello\r\n"},
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

// Large Value Tests
func TestRedis_LargeValues(t *testing.T) {
	db := setupTestDB()

	// Test with 1KB value
	largeValue := strings.Repeat("a", 1024)
	setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$9\r\nlarge_key\r\n$%d\r\n%s\r\n", len(largeValue), largeValue)
	result, err := executeCommand(db, setCmd)
	if err != nil {
		t.Fatalf("Failed to set large value: %v", err)
	}
	if result != "+OK\r\n" {
		t.Errorf("Expected OK for large value SET, got: %q", result)
	}

	// Retrieve large value
	getCmd := "*2\r\n$3\r\nGET\r\n$9\r\nlarge_key\r\n"
	result, err = executeCommand(db, getCmd)
	if err != nil {
		t.Fatalf("Failed to get large value: %v", err)
	}
	expected := fmt.Sprintf("+%s\r\n", largeValue)
	if result != expected {
		t.Errorf("Large value retrieval failed")
	}

	// Test with 10KB value
	veryLargeValue := strings.Repeat("b", 10*1024)
	setCmd = fmt.Sprintf("*3\r\n$3\r\nSET\r\n$14\r\nvery_large_key\r\n$%d\r\n%s\r\n", len(veryLargeValue), veryLargeValue)
	result, err = executeCommand(db, setCmd)
	if err != nil {
		t.Fatalf("Failed to set very large value: %v", err)
	}
	if result != "+OK\r\n" {
		t.Errorf("Expected OK for very large value SET, got: %q", result)
	}
}

// Special Characters Tests
func TestRedis_SpecialCharacters(t *testing.T) {
	db := setupTestDB()

	specialTests := []struct {
		name  string
		key   string
		value string
	}{
		{"unicode", "🔑", "🎯"},
		{"newlines", "key\nwith\nnewlines", "value\nwith\nnewlines"},
		{"tabs", "key\twith\ttabs", "value\twith\ttabs"},
		{"quotes", `key"with"quotes`, `value"with"quotes`},
		{"backslashes", `key\with\backslashes`, `value\with\backslashes`},
		{"mixed", "key🔑\n\t\"\\", "value🎯\n\t\"\\"},
	}

	for _, tt := range specialTests {
		t.Run(tt.name, func(t *testing.T) {
			// SET
			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n", len(tt.key), tt.key, len(tt.value), tt.value)
			result, err := executeCommand(db, setCmd)
			if err != nil {
				t.Fatalf("Failed to set special character key/value: %v", err)
			}
			if result != "+OK\r\n" {
				t.Errorf("Expected OK for special character SET, got: %q", result)
			}

			// GET
			getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(tt.key), tt.key)
			result, err = executeCommand(db, getCmd)
			if err != nil {
				t.Fatalf("Failed to get special character value: %v", err)
			}
			expected := fmt.Sprintf("+%s\r\n", tt.value)
			if result != expected {
				t.Errorf("Special character retrieval failed for %s", tt.name)
			}
		})
	}
}
