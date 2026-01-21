// Package ssh provides SSH key generation, validation, and path utilities.
package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath expands ~ and environment variables in a path.
// Returns the cleaned, absolute path.
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Expand environment variables first
	path = os.ExpandEnv(path)

	// Handle tilde expansion
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot expand ~: %w", err)
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}

	// Clean and return
	return filepath.Clean(path), nil
}

// DefaultSSHKeyPath returns the default SSH key path for a gitch identity.
// Format: ~/.ssh/gitch_{identityName}_ed25519
func DefaultSSHKeyPath(identityName string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Return empty on error - caller should handle
		return ""
	}
	return filepath.Join(home, ".ssh", fmt.Sprintf("gitch_%s_ed25519", identityName))
}
