package tests

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

func TestSimpleStringDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", "+\r\n"},
		{"Simple string", "OK", "+OK\r\n"},
		{"Pong", "PONG", "+PONG\r\n"},
		{"String with spaces", "Hello World", "+Hello World\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(resp.SimpleStringDecoder(tt.input))
			if result != tt.expected {
				t.Errorf("SimpleStringDecoder(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBulkStringDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Empty string", "", "$0\r\n\r\n"},
		{"Simple string", "hello", "$5\r\nhello\r\n"},
		{"String with spaces", "hello world", "$11\r\nhello world\r\n"},
		{"String with special chars", "hello\nworld", "$11\r\nhello\nworld\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(resp.BulkStringDecoder(tt.input))
			if result != tt.expected {
				t.Errorf("resp.BulkStringDecoder(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestErrorDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple error", "ERR unknown command", "-ERR unknown command\r\n"},
		{"Syntax error", "ERR syntax error", "-ERR syntax error\r\n"},
		{"Empty error", "", "-\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(resp.ErrorDecoder(tt.input))
			if result != tt.expected {
				t.Errorf("resp.ErrorDecoder(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIntegerDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"Zero", 0, ":0\r\n"},
		{"Positive", 42, ":42\r\n"},
		{"Negative", -123, ":-123\r\n"},
		{"Large number", 999999, ":999999\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(resp.IntegerDecoder(tt.input))
			if result != tt.expected {
				t.Errorf("resp.IntegerDecoder(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestArrayDecoder(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"Empty array", []string{}, "*0\r\n"},
		{"Single element", []string{"hello"}, "*1\r\n$5\r\nhello\r\n"},
		{"Two elements", []string{"SET", "key"}, "*2\r\n$3\r\nSET\r\n$3\r\nkey\r\n"},
		{"Three elements", []string{"SET", "key", "value"}, "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(resp.ArrayDecoder(tt.input))
			if result != tt.expected {
				t.Errorf("resp.ArrayDecoder(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmarks
func BenchmarkSimpleStringDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp.SimpleStringDecoder("OK")
	}
}

func BenchmarkBulkStringDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp.BulkStringDecoder("hello world")
	}
}

func BenchmarkErrorDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp.ErrorDecoder("ERR unknown command")
	}
}

func BenchmarkIntegerDecoder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resp.IntegerDecoder(42)
	}
}

func BenchmarkArrayDecoder(b *testing.B) {
	arr := []string{"SET", "key", "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp.ArrayDecoder(arr)
	}
}

func BenchmarkArrayDecoderLarge(b *testing.B) {
	arr := make([]string, 100)
	for i := range arr {
		arr[i] = "element"
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp.ArrayDecoder(arr)
	}
}
