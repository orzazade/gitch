package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestBuildDiffFields(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.name", "Current User")
	env.Run(t, "git", "config", "user.email", "current@example.com")

	identity := &config.Identity{
		Name:    "work",
		GitName: "Work User",
		Email:   "work@company.com",
	}

	fields, err := buildDiffFields(identity)
	if err != nil {
		t.Fatalf("buildDiffFields failed: %v", err)
	}

	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}

	// user.name should be changed
	nameField := fields[0]
	if nameField.Field != "user.name" {
		t.Errorf("expected field 'user.name', got %q", nameField.Field)
	}
	if nameField.Current != "Current User" {
		t.Errorf("expected current name 'Current User', got %q", nameField.Current)
	}
	if nameField.Target != "Work User" {
		t.Errorf("expected target name 'Work User', got %q", nameField.Target)
	}
	if !nameField.Changed {
		t.Error("expected user.name to be changed")
	}

	// user.email should be changed
	emailField := fields[1]
	if !emailField.Changed {
		t.Error("expected user.email to be changed")
	}
	if emailField.Current != "current@example.com" {
		t.Errorf("expected current email 'current@example.com', got %q", emailField.Current)
	}
	if emailField.Target != "work@company.com" {
		t.Errorf("expected target email 'work@company.com', got %q", emailField.Target)
	}
}

func TestBuildDiffFieldsExactMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.name", "Test User")
	env.Run(t, "git", "config", "user.email", "test@example.com")

	identity := &config.Identity{
		Name:    "test",
		GitName: "Test User",
		Email:   "test@example.com",
	}

	fields, err := buildDiffFields(identity)
	if err != nil {
		t.Fatalf("buildDiffFields failed: %v", err)
	}

	for _, f := range fields {
		if f.Changed {
			t.Errorf("expected field %q to be unchanged (current=%q target=%q)",
				f.Field, f.Current, f.Target)
		}
	}
}

func TestBuildDiffFieldsGPG(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.name", "GPG User")
	env.Run(t, "git", "config", "user.email", "gpg@example.com")

	identity := &config.Identity{
		Name:     "gpg",
		GitName:  "GPG User",
		Email:    "gpg@example.com",
		GPGKeyID: "ABC123DEF456",
	}

	fields, err := buildDiffFields(identity)
	if err != nil {
		t.Fatalf("buildDiffFields failed: %v", err)
	}

	// user.signingkey should change from empty to the key ID
	signingField := fields[3]
	if signingField.Field != "user.signingkey" {
		t.Errorf("expected field 'user.signingkey', got %q", signingField.Field)
	}
	if !signingField.Changed {
		t.Error("expected user.signingkey to be changed")
	}
	if signingField.Target != "ABC123DEF456" {
		t.Errorf("expected target signing key 'ABC123DEF456', got %q", signingField.Target)
	}

	// commit.gpgsign should change from empty to true
	gpgSignField := fields[4]
	if gpgSignField.Field != "commit.gpgsign" {
		t.Errorf("expected field 'commit.gpgsign', got %q", gpgSignField.Field)
	}
	if !gpgSignField.Changed {
		t.Error("expected commit.gpgsign to be changed")
	}
	if gpgSignField.Target != "true" {
		t.Errorf("expected target gpgsign 'true', got %q", gpgSignField.Target)
	}
}

func TestBuildDiffFieldsCaseInsensitiveEmail(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	env.Run(t, "git", "config", "user.name", "Test User")
	env.Run(t, "git", "config", "user.email", "Test@Example.COM")

	identity := &config.Identity{
		Name:    "test",
		GitName: "Test User",
		Email:   "test@example.com",
	}

	fields, err := buildDiffFields(identity)
	if err != nil {
		t.Fatalf("buildDiffFields failed: %v", err)
	}

	// Email should match case-insensitively
	emailField := fields[1]
	if emailField.Changed {
		t.Error("expected user.email to be unchanged (case-insensitive match)")
	}
}
