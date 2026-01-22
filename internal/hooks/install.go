// Package hooks provides pre-commit hook installation and management
package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/orzazade/gitch/internal/git"
)

// HooksDir returns the gitch hooks directory path
func HooksDir() (string, error) {
	return xdg.ConfigFile("gitch/hooks")
}

// InstallGlobal installs the pre-commit hook globally via core.hooksPath
func InstallGlobal() error {
	hooksDir, err := HooksDir()
	if err != nil {
		return fmt.Errorf("failed to determine hooks directory: %w", err)
	}

	// Create hooks directory
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write pre-commit script
	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte(PreCommitScript), 0755); err != nil {
		return fmt.Errorf("failed to write pre-commit hook: %w", err)
	}

	// Set git config --global core.hooksPath to hooksDir
	if err := git.SetConfig("core.hooksPath", hooksDir, true); err != nil {
		return fmt.Errorf("failed to set core.hooksPath: %w", err)
	}

	return nil
}

// UninstallGlobal removes the global hooks path configuration
func UninstallGlobal() error {
	// Unset git config --global core.hooksPath
	if err := git.UnsetConfig("core.hooksPath", true); err != nil {
		return fmt.Errorf("failed to unset core.hooksPath: %w", err)
	}

	// Optionally remove hooks directory
	hooksDir, err := HooksDir()
	if err != nil {
		return nil // Not critical if we can't determine the path
	}

	// Remove the hooks directory (ignore errors - user may have customized)
	_ = os.RemoveAll(hooksDir)

	return nil
}

// IsInstalled checks if gitch hooks are globally installed
func IsInstalled() (bool, error) {
	// Get current core.hooksPath value
	currentPath, err := git.GetConfig("core.hooksPath", true)
	if err != nil {
		return false, err
	}

	if currentPath == "" {
		return false, nil
	}

	// Check if it points to gitch hooks dir
	hooksDir, err := HooksDir()
	if err != nil {
		return false, err
	}

	// Compare paths (normalize for comparison)
	return filepath.Clean(currentPath) == filepath.Clean(hooksDir), nil
}
