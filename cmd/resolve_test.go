package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func TestClassifyQuery_DirectoryPaths(t *testing.T) {
	tests := []struct {
		query    string
		wantType string
	}{
		{"~/work/project", "directory"},
		{"/home/user/repos", "directory"},
		{"./relative/path", "directory"},
		{"../sibling", "directory"},
		{"somedir", "directory"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			gotType, cwd, remoteURL := classifyQuery(tt.query)
			if gotType != tt.wantType {
				t.Errorf("classifyQuery(%q) type = %q, want %q", tt.query, gotType, tt.wantType)
			}
			if cwd == "" {
				t.Errorf("classifyQuery(%q) cwd should not be empty for directory", tt.query)
			}
			if remoteURL != "" {
				t.Errorf("classifyQuery(%q) remoteURL = %q, want empty", tt.query, remoteURL)
			}
		})
	}
}

func TestClassifyQuery_RemoteURLs(t *testing.T) {
	tests := []struct {
		query    string
		wantType string
	}{
		{"git@github.com:company/repo.git", "remote"},
		{"https://github.com/personal/dotfiles.git", "remote"},
		{"ssh://git@gitlab.com/org/project.git", "remote"},
		{"github.com/org/repo", "remote"},
		{"gitlab.com/group/subgroup", "remote"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			gotType, cwd, remoteURL := classifyQuery(tt.query)
			if gotType != tt.wantType {
				t.Errorf("classifyQuery(%q) type = %q, want %q", tt.query, gotType, tt.wantType)
			}
			if remoteURL == "" {
				t.Errorf("classifyQuery(%q) remoteURL should not be empty for remote", tt.query)
			}
			if cwd != "" {
				t.Errorf("classifyQuery(%q) cwd = %q, want empty for remote", tt.query, cwd)
			}
		})
	}
}

func TestIsSCPStyle(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"git@github.com:org/repo.git", true},
		{"user@host:path", true},
		{"https://github.com/org/repo", false},
		{"github.com/org/repo", false},
		{"/home/user/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isSCPStyle(tt.input)
			if got != tt.want {
				t.Errorf("isSCPStyle(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunResolve_DirectoryMatch(t *testing.T) {
	env := setupResolveEnv(t)

	err := runResolve(resolveCmd, []string{env.matchDir})
	if err != nil {
		t.Fatalf("runResolve() error = %v, want nil", err)
	}
}

func TestRunResolve_RemoteMatch(t *testing.T) {
	setupResolveEnv(t)

	err := runResolve(resolveCmd, []string{"github.com/company/project"})
	if err != nil {
		t.Fatalf("runResolve() error = %v, want nil", err)
	}
}

func TestRunResolve_NoMatch(t *testing.T) {
	setupResolveEnv(t)

	err := runResolve(resolveCmd, []string{"/nonexistent/path/that/matches/nothing"})
	if err == nil {
		t.Fatal("runResolve() error = nil, want error for no match")
	}
}

func TestRunResolve_NoRules(t *testing.T) {
	setupResolveEnvNoRules(t)

	err := runResolve(resolveCmd, []string{"/some/path"})
	if err == nil {
		t.Fatal("runResolve() error = nil, want error for no rules")
	}
}

func TestRunResolve_SCPRemoteMatch(t *testing.T) {
	setupResolveEnv(t)

	err := runResolve(resolveCmd, []string{"git@github.com:company/project.git"})
	if err != nil {
		t.Fatalf("runResolve() error = %v, want nil", err)
	}
}

func TestRunResolve_HTTPSRemoteMatch(t *testing.T) {
	setupResolveEnv(t)

	err := runResolve(resolveCmd, []string{"https://github.com/company/project.git"})
	if err != nil {
		t.Fatalf("runResolve() error = %v, want nil", err)
	}
}

// --- test helpers ---

type resolveEnv struct {
	matchDir string
}

func setupResolveEnv(t *testing.T) *resolveEnv {
	t.Helper()

	matchDir := t.TempDir()

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@company.com"},
			{Name: "personal", GitName: "Jane", Email: "jane@gmail.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: matchDir, Identity: "work"},
			{Type: rules.RemoteRule, Pattern: "github.com/company/*", Identity: "work"},
			{Type: rules.RemoteRule, Pattern: "github.com/personal/*", Identity: "personal"},
		},
	}

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	t.Setenv("GITCH_CONFIG_PATH", configPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	resolveJSON = false

	return &resolveEnv{matchDir: matchDir}
}

func setupResolveEnvNoRules(t *testing.T) {
	t.Helper()

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@company.com"},
		},
		Rules: []rules.Rule{},
	}

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	t.Setenv("GITCH_CONFIG_PATH", configPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	resolveJSON = false
}
