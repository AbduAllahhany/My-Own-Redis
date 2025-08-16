package tests

import (
	"bufio"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/engine"
	"github.com/codecrafters-io/redis-starter-go/app/server"
)

// Simplified integration tests that don't use network connections

func TestRedisIntegration_BasicOperations(t *testing.T) {
	// Initialize commands
	server.InitCommands()

	// Create in-memory database
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "PING command",
			command:  "*1\r\n$4\r\nPING\r\n",
			expected: "+PONG\r\n",
		},
		{
			name:     "SET command",
			command:  "*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$6\r\nvalue1\r\n",
			expected: "+OK\r\n",
		},
		{
			name:     "GET command",
			command:  "*2\r\n$3\r\nGET\r\n$4\r\nkey1\r\n",
			expected: "+value1\r\n", // GET returns simple string in this implementation
		},
		{
			name:     "GET non-existing key",
			command:  "*2\r\n$3\r\nGET\r\n$7\r\nmissing\r\n",
			expected: "$-1\r\n",
		},
		{
			name:     "ECHO command",
			command:  "*2\r\n$4\r\nECHO\r\n$11\r\nhello world\r\n",
			expected: "$11\r\nhello world\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse command
			reader := bufio.NewReader(strings.NewReader(tt.command))
			cmd, err := server.ReadCommand(reader)
			if err != nil {
				t.Fatalf("Failed to parse command: %v", err)
			}

			// Create mock request (without network connection)
			request := &server.Request{
				Serv: &server.Server{
					Db: &db,
				},
				Cmd: &cmd,
			}

			// Process command
			result, err := server.ProcessCommand(request)
			if err != nil {
				t.Fatalf("Failed to process command: %v", err)
			}

			resultStr := string(result)
			if resultStr != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, resultStr)
			}
		})
	}
}

func TestRedisIntegration_SetWithExpiration(t *testing.T) {
	// Initialize commands
	server.InitCommands()

	// Create in-memory database
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	// Test SET with expiration
	reader := bufio.NewReader(strings.NewReader("*5\r\n$3\r\nSET\r\n$7\r\nexp_key\r\n$5\r\nvalue\r\n$2\r\nPX\r\n$3\r\n100\r\n"))
	cmd, err := server.ReadCommand(reader)
	if err != nil {
		t.Fatalf("Failed to parse SET command: %v", err)
	}

	request := &server.Request{
		Serv: &server.Server{
			Db: &db,
		},
		Cmd: &cmd,
	}

	result, err := server.ProcessCommand(request)
	if err != nil {
		t.Fatalf("Failed to process SET command: %v", err)
	}

	if string(result) != "+OK\r\n" {
		t.Errorf("Expected +OK, got %q", string(result))
	}

	// Verify key exists
	db.Mu.RLock()
	obj := (*db.Dict)["exp_key"]
	db.Mu.RUnlock()

	if obj == nil {
		t.Error("Expected key to exist")
	} else if !obj.HasExpiration() {
		t.Error("Expected key to have expiration")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify key is expired
	if obj.Value() != nil {
		t.Error("Expected key to be expired")
	}
}

func TestRedisIntegration_ConcurrentOperations(t *testing.T) {
	// Initialize commands
	server.InitCommands()

	// Create in-memory database
	dict := make(map[string]engine.RedisObj)
	db := engine.DbStore{
		Dict: &dict,
		Mu:   &sync.RWMutex{},
	}

	var wg sync.WaitGroup
	numGoroutines := 5
	numOperations := 10

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				key := "key" + string(rune(id+48)) + string(rune(j+48))
				value := "value" + string(rune(j+48))

				// SET operation
				setCmd := "*3\r\n$3\r\nSET\r\n$" + string(rune(len(key)+48)) + "\r\n" + key + "\r\n$" + string(rune(len(value)+48)) + "\r\n" + value + "\r\n"
				reader := bufio.NewReader(strings.NewReader(setCmd))
				cmd, err := server.ReadCommand(reader)
				if err != nil {
					t.Errorf("Failed to parse SET command: %v", err)
					return
				}

				request := &server.Request{
					Serv: &server.Server{
						Db: &db,
					},
					Cmd: &cmd,
				}

				_, err = server.ProcessCommand(request)
				if err != nil {
					t.Errorf("Failed to process SET command: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state
	db.Mu.RLock()
	totalKeys := len(*db.Dict)
	db.Mu.RUnlock()

	expectedKeys := numGoroutines * numOperations
	if totalKeys != expectedKeys {
		t.Errorf("Expected %d keys, got %d", expectedKeys, totalKeys)
	}
}
