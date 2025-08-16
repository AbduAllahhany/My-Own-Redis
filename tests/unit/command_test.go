package tests

import (
	"bufio"
	"strings"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

func TestReadCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    server.Command
		shouldError bool
	}{
		{
			name:  "Valid SET command",
			input: "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
			expected: server.Command{
				Name: "SET",
				Args: []string{"key", "value"},
			},
		},
		{
			name:  "Valid GET command",
			input: "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n",
			expected: server.Command{
				Name: "GET",
				Args: []string{"key"},
			},
		},
		{
			name:  "Valid PING command",
			input: "*1\r\n$4\r\nPING\r\n",
			expected: server.Command{
				Name: "PING",
				Args: nil,
			},
		},
		{
			name:        "Invalid format - not array",
			input:       "+OK\r\n",
			shouldError: true,
		},
		{
			name:        "Empty array",
			input:       "*0\r\n",
			shouldError: true,
		},
	}

	// Initialize commands for testing
	server.InitCommands()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			result, err := server.ReadCommand(reader)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Name != tt.expected.Name {
				t.Errorf("Expected name %s, got %s", tt.expected.Name, result.Name)
			}

			if len(result.Args) != len(tt.expected.Args) {
				t.Errorf("Expected %d args, got %d", len(tt.expected.Args), len(result.Args))
				return
			}

			for i, arg := range tt.expected.Args {
				if result.Args[i] != arg {
					t.Errorf("Expected arg[%d] %s, got %s", i, arg, result.Args[i])
				}
			}
		})
	}
}

// Benchmark tests
func BenchmarkReadCommand(b *testing.B) {
	server.InitCommands()
	input := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bufio.NewReader(strings.NewReader(input))
		_, _ = server.ReadCommand(reader)
	}
}
