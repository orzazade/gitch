// Package hooks provides pre-commit hook installation and management
package hooks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/orzazade/gitch/internal/git"
)

// HooksDir returns the gitch-managed global hooks directory path.
func HooksDir() (string, error) {
	return xdg.ConfigFile("gitch/hooks")
}

// localHookPath returns the current repository's pre-commit hook path.
func localHookPath() (string, error) {
	return git.GitPath(filepath.Join("hooks", "pre-commit"))
}

// localPrePushPath returns the current repository's pre-push hook path.
func localPrePushPath() (string, error) {
	return git.GitPath(filepath.Join("hooks", "pre-push"))
}

// InstallLocal installs the pre-commit hook in the current repository.
func InstallLocal() error {
	preCommitPath, err := localHookPath()
	if err != nil {
		return err
	}

	if err := ensureHookPathWritable(preCommitPath); err != nil {
		return err
	}

	return writeManagedHook(preCommitPath)
}

// UninstallLocal removes the gitch pre-commit hook from the current repository.
func UninstallLocal() error {
	preCommitPath, err := localHookPath()
	if err != nil {
		return err
	}

	return removeManagedHook(preCommitPath)
}

// IsInstalledLocal checks whether the current repository has the gitch hook installed.
func IsInstalledLocal() (bool, error) {
	preCommitPath, err := localHookPath()
	if err != nil {
		return false, err
	}
	return isManagedHook(preCommitPath)
}

// InstallLocalPrePush installs the pre-push hook in the current repository.
func InstallLocalPrePush() error {
	prePushPath, err := localPrePushPath()
	if err != nil {
		return err
	}

	if err := ensureHookPathWritable(prePushPath); err != nil {
		return err
	}

	return writeHook(prePushPath, prePushScript)
}

// UninstallLocalPrePush removes the gitch pre-push hook from the current repository.
func UninstallLocalPrePush() error {
	prePushPath, err := localPrePushPath()
	if err != nil {
		return err
	}

	return removeManagedHook(prePushPath)
}

// IsInstalledLocalPrePush checks whether the current repository has the gitch pre-push hook.
func IsInstalledLocalPrePush() (bool, error) {
	prePushPath, err := localPrePushPath()
	if err != nil {
		return false, err
	}
	return isManagedHook(prePushPath)
}

// resolveGlobalHookPaths returns the current core.hooksPath and the gitch-managed hooks directory.
func resolveGlobalHookPaths() (currentPath, hooksDir string, err error) {
	currentPath, err = git.GetConfigScoped("core.hooksPath", git.ScopeGlobal)
	if err != nil {
		return "", "", fmt.Errorf("failed to read core.hooksPath: %w", err)
	}
	hooksDir, err = HooksDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to determine hooks directory: %w", err)
	}
	return currentPath, hooksDir, nil
}

// InstallGlobal installs the pre-commit hook globally via core.hooksPath.
// It refuses to overwrite an existing non-gitch global hooksPath.
func InstallGlobal() error {
	currentPath, hooksDir, err := resolveGlobalHookPaths()
	if err != nil {
		return err
	}

	if currentPath != "" && filepath.Clean(currentPath) != filepath.Clean(hooksDir) {
		return fmt.Errorf("core.hooksPath is already set to %s; refusing to overwrite existing global hooks", currentPath)
	}

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := ensureHookPathWritable(preCommitPath); err != nil {
		return err
	}
	if err := writeManagedHook(preCommitPath); err != nil {
		return err
	}

	prePushPath := filepath.Join(hooksDir, "pre-push")
	if err := ensureHookPathWritable(prePushPath); err != nil {
		return err
	}
	if err := writeHook(prePushPath, prePushScript); err != nil {
		return err
	}

	if err := git.SetConfigScoped("core.hooksPath", hooksDir, git.ScopeGlobal); err != nil {
		return fmt.Errorf("failed to set core.hooksPath: %w", err)
	}

	return nil
}

// UninstallGlobal removes the gitch global hook if gitch owns core.hooksPath.
func UninstallGlobal() error {
	currentPath, hooksDir, err := resolveGlobalHookPaths()
	if err != nil {
		return err
	}

	if filepath.Clean(currentPath) != filepath.Clean(hooksDir) {
		return nil
	}

	if err := git.UnsetConfigScoped("core.hooksPath", git.ScopeGlobal); err != nil {
		return fmt.Errorf("failed to unset core.hooksPath: %w", err)
	}

	_ = removeManagedHook(filepath.Join(hooksDir, "pre-commit"))
	_ = removeManagedHook(filepath.Join(hooksDir, "pre-push"))
	_ = os.Remove(hooksDir)
	return nil
}

// IsInstalled checks if gitch hooks are globally installed.
func IsInstalled() (bool, error) {
	currentPath, hooksDir, err := resolveGlobalHookPaths()
	if err != nil {
		return false, err
	}

	if currentPath == "" || filepath.Clean(currentPath) != filepath.Clean(hooksDir) {
		return false, nil
	}

	return isManagedHook(filepath.Join(hooksDir, "pre-commit"))
}

func ensureHookPathWritable(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create hook directory: %w", err)
	}

	managed, err := isManagedHook(path)
	if err != nil {
		return err
	}
	if managed {
		return nil
	}

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("hook already exists at %s and is not managed by gitch", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to inspect hook %s: %w", path, err)
	}

	return nil
}

func writeManagedHook(path string) error {
	return writeHook(path, preCommitScript)
}

func writeHook(path string, script string) error {
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write hook %s: %w", filepath.Base(path), err)
	}
	return nil
}

func removeManagedHook(path string) error {
	managed, err := isManagedHook(path)
	if err != nil {
		return err
	}
	if !managed {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove hook %s: %w", path, err)
	}
	return nil
}

func isManagedHook(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read hook %s: %w", path, err)
	}
	return strings.Contains(string(data), managedHookMarker), nil
}
