package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestResolveCurrentProfileState_ExactMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
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
	defer env.Cleanup(t)
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
