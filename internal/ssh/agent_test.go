package ssh

import (
	"os"
	"testing"
)

func TestIsAgentRunning(t *testing.T) {
	// This test depends on the environment - it will pass if ssh-agent is running
	// and fail if it's not. We test both code paths.
	socket := os.Getenv("SSH_AUTH_SOCK")

	t.Run("matches environment", func(t *testing.T) {
		result := IsAgentRunning()

		if socket == "" {
			// No SSH_AUTH_SOCK set - should return false
			if result {
				t.Error("IsAgentRunning() returned true when SSH_AUTH_SOCK is not set")
			}
		} else {
			// SSH_AUTH_SOCK is set - result depends on whether the socket is reachable
			// Just verify we don't panic
			t.Logf("IsAgentRunning() = %v (SSH_AUTH_SOCK=%s)", result, socket)
		}
	})
}

func TestIsAgentRunning_NoSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	if IsAgentRunning() {
		t.Error("IsAgentRunning() returned true when SSH_AUTH_SOCK is empty")
	}
}

func TestIsAgentRunning_InvalidSocket(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "/nonexistent/path/to/socket")

	if IsAgentRunning() {
		t.Error("IsAgentRunning() returned true for invalid socket path")
	}
}

func TestAddKeyToAgent_NoAgent(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")

	err := AddKeyToAgent("/some/key/path")
	if err == nil {
		t.Fatal("AddKeyToAgent() should return error when agent not running")
	}

	expected := "ssh-agent not running. Start it with: eval $(ssh-agent)"
	if err.Error() != expected {
		t.Errorf("AddKeyToAgent() error = %q, want %q", err.Error(), expected)
	}
}
