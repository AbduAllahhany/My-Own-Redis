package tests

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

func TestCommandInitialization(t *testing.T) {
	// Test that commands can be initialized
	server.InitCommands()

	// Test that we can create commands
	cmd := server.CreateCommand("PING", []string{})
	if cmd.Name != "PING" {
		t.Errorf("Expected command name 'PING', got '%s'", cmd.Name)
	}

	if cmd.Handle == nil {
		t.Error("Expected command handler to be set")
	}
}

func TestServerConfiguration(t *testing.T) {
	config := server.NewConfiguration()
	if config.Port == "" {
		t.Error("Port should not be empty")
	}
	if config.Dir == "" {
		t.Error("Dir should not be empty")
	}
}
