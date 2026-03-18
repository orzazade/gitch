package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestResolveApplyScope_BothFlags(t *testing.T) {
	_, err := resolveApplyScope(true, true)
	if err == nil {
		t.Fatal("expected error when both --local and --global are set")
	}
}

func TestResolveApplyScope_LocalOnly(t *testing.T) {
	scope, err := resolveApplyScope(true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != git.ScopeLocal {
		t.Errorf("expected ScopeLocal, got %q", scope)
	}
}

func TestResolveApplyScope_GlobalOnly(t *testing.T) {
	scope, err := resolveApplyScope(false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != git.ScopeGlobal {
		t.Errorf("expected ScopeGlobal, got %q", scope)
	}
}

func TestResolveApplyScope_NeitherFlag(t *testing.T) {
	// When neither flag is set, should return a valid scope without error
	scope, err := resolveApplyScope(false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope != git.ScopeLocal && scope != git.ScopeGlobal {
		t.Errorf("expected ScopeLocal or ScopeGlobal, got %q", scope)
	}
}

func TestShortHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abcdef1234567890", "abcdef12"},
		{"abcdef12", "abcdef12"},
		{"abc", "abc"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := shortHash(tt.input); got != tt.want {
			t.Errorf("shortHash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncateSubject(t *testing.T) {
	tests := []struct {
		subject string
		maxLen  int
		want    string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a longer subject line", 20, "this is a longer ..."},
		{"", 10, ""},
		// Multi-byte UTF-8: should truncate by rune count, not byte count
		{"日本語のコミットメッセージです", 10, "日本語のコミッ..."},
		{"café résumé", 8, "café ..."},
	}
	for _, tt := range tests {
		if got := truncateSubject(tt.subject, tt.maxLen); got != tt.want {
			t.Errorf("truncateSubject(%q, %d) = %q, want %q", tt.subject, tt.maxLen, got, tt.want)
		}
	}
}

func TestResolveCurrentProfileState_ExactMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{
				Name:    "work",
				GitName: "Jane Doe",
				Email:   "jane@example.com",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	if err := git.ApplyIdentityScoped("Jane Doe", "jane@example.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply local identity: %v", err)
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		t.Fatalf("resolveCurrentProfileState failed: %v", err)
	}
	if state.ExactMatch == nil || state.ExactMatch.Name != "work" {
		t.Fatalf("expected exact match 'work', got %#v", state.ExactMatch)
	}
	if state.EmailMatch == nil || state.EmailMatch.Name != "work" {
		t.Fatalf("expected email match 'work', got %#v", state.EmailMatch)
	}
}

func TestResolveCurrentProfileState_PartialEmailMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{
				Name:    "work",
				GitName: "Jane Doe",
				Email:   "jane@example.com",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	if err := git.ApplyIdentityScoped("Wrong Name", "jane@example.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply local identity: %v", err)
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		t.Fatalf("resolveCurrentProfileState failed: %v", err)
	}
	if state.ExactMatch != nil {
		t.Fatalf("expected no exact match, got %#v", state.ExactMatch)
	}
	if state.EmailMatch == nil || state.EmailMatch.Name != "work" {
		t.Fatalf("expected email match 'work', got %#v", state.EmailMatch)
	}
}
