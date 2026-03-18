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

// Scope controls which git configuration scope is read or written.
type Scope string

const (
	ScopeEffective Scope = "effective"
	ScopeGlobal    Scope = "global"
	ScopeLocal     Scope = "local"
)

func scopeArgs(scope Scope) ([]string, error) {
	switch scope {
	case ScopeEffective:
		return []string{"config"}, nil
	case ScopeGlobal:
		return []string{"config", "--global"}, nil
	case ScopeLocal:
		return []string{"config", "--local"}, nil
	default:
		return nil, fmt.Errorf("unknown git config scope %q", scope)
	}
}

// GetConfigScoped reads a git config value from the requested scope.
// Effective scope reads the value that git would currently use.
func GetConfigScoped(key string, scope Scope) (string, error) {
	args, err := scopeArgs(scope)
	if err != nil {
		return "", err
	}

	args = append(args, "--get", key)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", ErrGitNotFound
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", nil
		}

		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// SetConfigScoped writes a git config value to the requested scope.
func SetConfigScoped(key, value string, scope Scope) error {
	args, err := scopeArgs(scope)
	if err != nil {
		return err
	}

	args = append(args, key, value)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
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
	scope := ScopeLocal
	if global {
		scope = ScopeGlobal
	}
	return UnsetConfigScoped(key, scope)
}

// UnsetConfigScoped removes a git config key from the requested scope.
func UnsetConfigScoped(key string, scope Scope) error {
	args, err := scopeArgs(scope)
	if err != nil {
		return err
	}

	args = append(args, "--unset", key)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return ErrGitNotFound
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 5 || exitErr.ExitCode() == 1) {
			return nil
		}

		return fmt.Errorf("failed to unset git config %s: %w", key, err)
	}

	return nil
}

// GetCurrentIdentity returns the effective git user.name and user.email for the
// current working directory, falling back to global config outside repositories.
func GetCurrentIdentity() (name string, email string, err error) {
	return GetCurrentIdentityScoped(ScopeEffective)
}

// GetCurrentIdentityScoped reads git user.name and user.email from a specific scope.
func GetCurrentIdentityScoped(scope Scope) (name string, email string, err error) {
	name, err = GetConfigScoped("user.name", scope)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.name: %w", err)
	}

	email, err = GetConfigScoped("user.email", scope)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user.email: %w", err)
	}

	return name, email, nil
}

// ApplyIdentityScoped sets git user.name and user.email in the requested scope.
func ApplyIdentityScoped(name, email string, scope Scope) error {
	if err := SetConfigScoped("user.name", name, scope); err != nil {
		return err
	}
	return SetConfigScoped("user.email", email, scope)
}

// ApplySigningConfigScoped configures git to use the specified GPG key for signing commits.
func ApplySigningConfigScoped(keyID string, scope Scope) error {
	if err := SetConfigScoped("user.signingkey", keyID, scope); err != nil {
		return fmt.Errorf("failed to set signing key: %w", err)
	}

	if err := SetConfigScoped("commit.gpgsign", "true", scope); err != nil {
		return fmt.Errorf("failed to enable commit signing: %w", err)
	}

	return nil
}

// ClearSigningConfigScoped removes GPG signing configuration from git config.
func ClearSigningConfigScoped(scope Scope) error {
	if err := UnsetConfigScoped("user.signingkey", scope); err != nil {
		return fmt.Errorf("failed to unset signing key: %w", err)
	}

	if err := UnsetConfigScoped("commit.gpgsign", scope); err != nil {
		return fmt.Errorf("failed to unset commit signing: %w", err)
	}

	return nil
}

// IsGitRepository reports whether the current working directory is inside a git repository.
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// GitPath resolves a path relative to the current repository's git directory.
func GitPath(relPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-path", relPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve git path %s: %w", relPath, err)
	}
	return strings.TrimSpace(string(output)), nil
}
