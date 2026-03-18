package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// GitEnv provides an isolated git environment for integration-style tests.
type GitEnv struct {
	Dir          string
	GlobalConfig string
	ConfigPath   string
}

// SetupGitEnv creates an isolated HOME, XDG config home, global git config, and git repo.
// Environment variables and working directory are automatically restored via t.Setenv/t.Chdir.
func SetupGitEnv(t *testing.T) *GitEnv {
	t.Helper()

	dir := t.TempDir()
	globalConfig := filepath.Join(dir, ".gitconfig")
	if err := os.WriteFile(globalConfig, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create global git config: %v", err)
	}

	configPath := filepath.Join(dir, "xdg", "gitch", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}

	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("GITCH_CONFIG_PATH", configPath)

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	return &GitEnv{
		Dir:          dir,
		GlobalConfig: globalConfig,
		ConfigPath:   configPath,
	}
}

// Chdir moves into the test repository.
// The original working directory is automatically restored when the test completes.
func (e *GitEnv) Chdir(t *testing.T) {
	t.Helper()
	t.Chdir(e.Dir)
}

// Run executes a command in the test repository.
func (e *GitEnv) Run(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = e.Dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %s %v failed: %v\n%s", name, args, err, string(output))
	}
	return string(output)
}
