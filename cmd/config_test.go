package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/testutil"
)

func captureConfigDefaultOutput(t *testing.T, args []string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := configDefaultCmd
	err := cmd.RunE(cmd, args)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func captureConfigPathOutput(t *testing.T) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := configPathCmd
	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String(), err
}

func TestConfigDefault_ShowNoDefault(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureConfigDefaultOutput(t, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "No default identity set") {
		t.Errorf("expected no-default message, got: %s", output)
	}
}

func TestConfigDefault_ShowExisting(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Default: "work",
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureConfigDefaultOutput(t, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(output) != "work" {
		t.Errorf("expected 'work', got: %q", strings.TrimSpace(output))
	}
}

func TestConfigDefault_SetValid(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "personal", Email: "personal@gmail.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	output, err := captureConfigDefaultOutput(t, []string{"personal"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "personal") {
		t.Errorf("expected success message with 'personal', got: %s", output)
	}

	// Verify it was persisted
	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if reloaded.Default != "personal" {
		t.Errorf("expected default 'personal', got: %q", reloaded.Default)
	}
}

func TestConfigDefault_SetCaseInsensitive(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "Work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, err := captureConfigDefaultOutput(t, []string{"work"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reloaded, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	// SetDefault preserves the original case from the stored identity
	if reloaded.Default != "Work" {
		t.Errorf("expected default 'Work' (original case), got: %q", reloaded.Default)
	}
}

func TestConfigDefault_SetNonexistent(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	_, err := captureConfigDefaultOutput(t, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent identity, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestConfigPath_ReturnsPath(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	output, err := captureConfigPathOutput(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := strings.TrimSpace(output)
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if path != env.ConfigPath {
		t.Errorf("expected %q, got: %q", env.ConfigPath, path)
	}
}
