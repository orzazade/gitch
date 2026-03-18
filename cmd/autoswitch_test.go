package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestFormatCurrentIdentity(t *testing.T) {
	tests := []struct {
		name     string
		gitName  string
		email    string
		expected string
	}{
		{"both set", "Jane Doe", "jane@example.com", "Jane Doe <jane@example.com>"},
		{"name only", "Jane Doe", "", "Jane Doe"},
		{"email only", "", "jane@example.com", "jane@example.com"},
		{"neither set", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCurrentIdentity(tt.gitName, tt.email)
			if got != tt.expected {
				t.Errorf("formatCurrentIdentity(%q, %q) = %q, want %q", tt.gitName, tt.email, got, tt.expected)
			}
		})
	}
}

func TestTryAutoSwitch_UsesLocalScopeInsideRepo(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	keyPath := filepath.Join(env.Dir, "id_test")
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0600); err != nil {
		t.Fatalf("failed to create SSH key fixture: %v", err)
	}

	cfg := &config.Config{
		Identities: []config.Identity{
			{
				Name:       "work",
				GitName:    "Jane Doe",
				Email:      "jane@example.com",
				SSHKeyPath: keyPath,
				GPGKeyID:   "ABC123",
			},
		},
		Rules: []rules.Rule{
			{
				Type:     rules.DirectoryRule,
				Pattern:  env.Dir,
				Identity: "work",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	result, err := TryAutoSwitch(cfg)
	if err != nil {
		t.Fatalf("TryAutoSwitch failed: %v", err)
	}
	if result == nil || !result.Switched {
		t.Fatalf("expected TryAutoSwitch to switch, got %#v", result)
	}

	localEmail, err := git.GetConfigScoped("user.email", git.ScopeLocal)
	if err != nil {
		t.Fatalf("failed to read local user.email: %v", err)
	}
	if localEmail != "jane@example.com" {
		t.Fatalf("local user.email = %q, want %q", localEmail, "jane@example.com")
	}

	globalEmail, err := git.GetConfigScoped("user.email", git.ScopeGlobal)
	if err != nil {
		t.Fatalf("failed to read global user.email: %v", err)
	}
	if globalEmail != "" {
		t.Fatalf("global user.email = %q, want empty", globalEmail)
	}
}
