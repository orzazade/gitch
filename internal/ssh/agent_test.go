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
	// Save and clear the socket
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		}
	}()

	if IsAgentRunning() {
		t.Error("IsAgentRunning() returned true when SSH_AUTH_SOCK is unset")
	}
}

func TestIsAgentRunning_InvalidSocket(t *testing.T) {
	// Save and set an invalid socket
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", "/nonexistent/path/to/socket")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	if IsAgentRunning() {
		t.Error("IsAgentRunning() returned true for invalid socket path")
	}
}

func TestAddKeyToAgent_NoAgent(t *testing.T) {
	// Save and clear the socket
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		}
	}()

	err := AddKeyToAgent("/some/key/path")
	if err == nil {
		t.Fatal("AddKeyToAgent() should return error when agent not running")
	}

	expected := "ssh-agent not running. Start it with: eval $(ssh-agent)"
	if err.Error() != expected {
		t.Errorf("AddKeyToAgent() error = %q, want %q", err.Error(), expected)
	}
}

func TestAddKeyToAgentWithPassphrase_NoAgent(t *testing.T) {
	// Save and clear the socket
	original := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer func() {
		if original != "" {
			os.Setenv("SSH_AUTH_SOCK", original)
		}
	}()

	err := AddKeyToAgentWithPassphrase("/some/key/path", nil)
	if err == nil {
		t.Fatal("AddKeyToAgentWithPassphrase() should return error when agent not running")
	}

	expected := "ssh-agent not running. Start it with: eval $(ssh-agent)"
	if err.Error() != expected {
		t.Errorf("AddKeyToAgentWithPassphrase() error = %q, want %q", err.Error(), expected)
	}
}

func TestAddKeyToAgentWithPassphrase_KeyNotFound(t *testing.T) {
	// Skip if no agent running
	if !IsAgentRunning() {
		t.Skip("ssh-agent not running, skipping test")
	}

	err := AddKeyToAgentWithPassphrase("/nonexistent/key/path", nil)
	if err == nil {
		t.Fatal("AddKeyToAgentWithPassphrase() should return error for nonexistent key")
	}

	// Should contain "failed to read key file"
	if err.Error()[:22] != "failed to read key fil" {
		t.Errorf("AddKeyToAgentWithPassphrase() error = %q, want to contain 'failed to read key file'", err.Error())
	}
}

func TestAddKeyToAgentWithPassphrase_ValidKey(t *testing.T) {
	// Skip if no agent running
	if !IsAgentRunning() {
		t.Skip("ssh-agent not running, skipping test")
	}

	// Create a temporary key file (unencrypted)
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/test_key"

	priv, _, err := GenerateKeyPair("test@example.com", nil)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Add to agent
	err = AddKeyToAgentWithPassphrase(keyPath, nil)
	if err != nil {
		t.Errorf("AddKeyToAgentWithPassphrase() error = %v", err)
	}
}
