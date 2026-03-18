package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// errAgentNotRunning is returned when ssh-agent is not running or not accessible.
var errAgentNotRunning = errors.New("ssh-agent not running. Start it with: eval $(ssh-agent)")

// IsAgentRunning checks if ssh-agent is running and accessible.
// Returns true if SSH_AUTH_SOCK is set and the socket is reachable.
func IsAgentRunning() bool {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return false
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// AddKeyToAgent adds an SSH key to the running ssh-agent.
// On macOS, uses /usr/bin/ssh-add with --apple-use-keychain for Keychain integration.
// On other platforms, uses standard ssh-add.
// This method uses exec to shell out, allowing passphrase prompts to work interactively.
func AddKeyToAgent(keyPath string) error {
	if !IsAgentRunning() {
		return errAgentNotRunning
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// macOS: Use system ssh-add with Keychain integration
		// CRITICAL: Use full path /usr/bin/ssh-add to avoid Homebrew's ssh-add
		// which may not support --apple-use-keychain
		cmd = exec.Command("/usr/bin/ssh-add", "--apple-use-keychain", keyPath)
	} else {
		// Linux and other platforms: Use standard ssh-add
		cmd = exec.Command("ssh-add", keyPath)
	}

	// Connect stdin/stdout/stderr for interactive passphrase prompt
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// IsKeyLoadedInAgent checks whether the SSH key at keyPath is loaded in the
// running ssh-agent by comparing public key fingerprints. Returns false if
// the agent is not running, the public key file cannot be read, or the key
// is not found in the agent.
func IsKeyLoadedInAgent(keyPath string) bool {
	if !IsAgentRunning() {
		return false
	}

	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return false
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyData)
	if err != nil {
		return false
	}

	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return false
	}
	defer conn.Close()

	keys, err := agent.NewClient(conn).List()
	if err != nil {
		return false
	}

	target := ssh.FingerprintSHA256(pubKey)
	for _, k := range keys {
		if ssh.FingerprintSHA256(k) == target {
			return true
		}
	}
	return false
}

// AddKeyToAgentWithPassphrase adds an SSH key to the agent programmatically.
// If passphrase is nil or empty and the key requires one, falls back to AddKeyToAgent
// to allow interactive passphrase prompting.
func AddKeyToAgentWithPassphrase(keyPath string, passphrase []byte) error {
	if !IsAgentRunning() {
		return errAgentNotRunning
	}

	socket := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to connect to ssh-agent: %w", err)
	}
	defer conn.Close()

	// Read key file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse key
	var privKey any
	if len(passphrase) > 0 {
		privKey, err = ssh.ParseRawPrivateKeyWithPassphrase(keyData, passphrase)
	} else {
		privKey, err = ssh.ParseRawPrivateKey(keyData)
	}
	if err != nil {
		// If the key needs a passphrase, fall back to shell method for interactive prompt
		var passErr *ssh.PassphraseMissingError
		if errors.As(err, &passErr) {
			return AddKeyToAgent(keyPath)
		}
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Add to agent
	agentClient := agent.NewClient(conn)
	return agentClient.Add(agent.AddedKey{
		PrivateKey: privKey,
		Comment:    filepath.Base(keyPath),
	})
}
