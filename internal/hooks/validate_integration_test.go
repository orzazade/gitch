package hooks

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestValidate_NoRuleMatches(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Save config with no rules
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	result, err := Validate()
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Match {
		t.Error("expected Match=true when no rule matches")
	}
}

func TestValidate_MatchingIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Save config with identity and directory rule
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Apply the expected identity
	if err := git.ApplyIdentityScoped("Jane Doe", "jane@work.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	result, err := Validate()
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Match {
		t.Error("expected Match=true when identity matches rule")
	}
	if result.ExpectedEmail != "jane@work.com" {
		t.Errorf("expected email jane@work.com, got %q", result.ExpectedEmail)
	}
}

func TestValidate_UnknownRuleIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	// Rule references an identity that doesn't exist
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "nonexistent"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	_, err := Validate()
	if err == nil {
		t.Fatal("expected error when rule references unknown identity")
	}
}

func TestValidate_MismatchedIdentity(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "jane@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Apply a DIFFERENT identity
	if err := git.ApplyIdentityScoped("Personal", "personal@home.com", git.ScopeLocal); err != nil {
		t.Fatalf("failed to apply identity: %v", err)
	}

	result, err := Validate()
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Match {
		t.Error("expected Match=false when identity doesn't match rule")
	}
	if result.CurrentEmail != "personal@home.com" {
		t.Errorf("expected current email personal@home.com, got %q", result.CurrentEmail)
	}
}
