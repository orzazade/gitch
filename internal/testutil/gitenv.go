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
	origEnv      map[string]string
	origDir      string
}

// SetupGitEnv creates an isolated HOME, XDG config home, global git config, and git repo.
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

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	origEnv := map[string]string{}
	for _, key := range []string{"GIT_CONFIG_GLOBAL", "HOME", "XDG_CONFIG_HOME", "GITCH_CONFIG_PATH"} {
		origEnv[key] = os.Getenv(key)
	}

	os.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	os.Setenv("HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	os.Setenv("GITCH_CONFIG_PATH", configPath)

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	return &GitEnv{
		Dir:          dir,
		GlobalConfig: globalConfig,
		ConfigPath:   configPath,
		origEnv:      origEnv,
		origDir:      origDir,
	}
}

// Cleanup restores the original environment and working directory.
func (e *GitEnv) Cleanup(t *testing.T) {
	t.Helper()

	for key, value := range e.origEnv {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}

	if err := os.Chdir(e.origDir); err != nil {
		t.Fatalf("failed to restore working directory: %v", err)
	}
}

// Chdir moves into the test repository.
func (e *GitEnv) Chdir(t *testing.T) {
	t.Helper()
	if err := os.Chdir(e.Dir); err != nil {
		t.Fatalf("failed to change directory to test repo: %v", err)
	}
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
