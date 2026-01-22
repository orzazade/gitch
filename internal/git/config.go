// Package git provides an adapter for reading and writing git configuration.
package git

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrGitNotFound indicates git binary was not found on the system.
var ErrGitNotFound = errors.New("git: executable not found in PATH")

// GetConfig reads a git config value.
// If global is true, reads from --global scope; otherwise reads from local repo.
// Returns empty string if key is not set (not an error).
func GetConfig(key string, global bool) (string, error) {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, "--get", key)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if git is not found
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrGitNotFound
		}

		// Exit code 1 means key not set - this is not an error, just return empty
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}

		// Other errors should be wrapped with context
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// SetConfig writes a git config value.
// If global is true, writes to --global scope; otherwise writes to local repo.
func SetConfig(key, value string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		// Check if git is not found
		if errors.Is(err, exec.ErrNotFound) {
			return ErrGitNotFound
		}
		return fmt.Errorf("failed to set git config %s: %w", key, err)
	}

	return nil
}

// UnsetConfig removes a git config key.
// If global is true, removes from --global scope; otherwise removes from local repo.
// Returns nil if the key was not set (idempotent).
func UnsetConfig(key string, global bool) error {
	args := []string{"config", "--unset"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		// Check if git is not found
		if errors.Is(err, exec.ErrNotFound) {
			return ErrGitNotFound
		}

		// Exit code 5 means key was not set - this is not an error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 5 {
			return nil
		}

		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}

	return nil
}

// GetCurrentIdentity returns the current git user.name and user.email from global config.
// Either value may be empty if not set.
func GetCurrentIdentity() (name string, email string, err error) {
	name, err = GetConfig("user.name", true)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.name: %w", err)
	}

	email, err = GetConfig("user.email", true)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.email: %w", err)
	}

	return name, email, nil
}

// ApplyIdentity sets git user.name and user.email globally.
// Returns the first error encountered, if any.
func ApplyIdentity(name, email string) error {
	if err := SetConfig("user.name", name, true); err != nil {
		return fmt.Errorf("failed to apply identity: %w", err)
	}

	if err := SetConfig("user.email", email, true); err != nil {
		return fmt.Errorf("failed to apply identity: %w", err)
	}

	return nil
}
