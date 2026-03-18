package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestUpdateIdentity_Email(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "old@company.com"},
			{Name: "personal", GitName: "Jane Doe", Email: "jane@gmail.com"},
		},
	}

	newEmail := "new@company.com"
	updates := config.IdentityUpdates{Email: &newEmail}
	identity, err := cfg.UpdateIdentity("work", updates)
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if identity.Email != "new@company.com" {
		t.Errorf("expected email 'new@company.com', got %q", identity.Email)
	}
	if identity.GitName != "Jane Doe" {
		t.Errorf("expected git name unchanged 'Jane Doe', got %q", identity.GitName)
	}
}

func TestUpdateIdentity_DuplicateEmail(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "work@company.com"},
			{Name: "personal", GitName: "Jane Doe", Email: "jane@gmail.com"},
		},
	}

	dupEmail := "jane@gmail.com"
	updates := config.IdentityUpdates{Email: &dupEmail}
	_, err := cfg.UpdateIdentity("work", updates)
	if err == nil {
		t.Fatal("expected error for duplicate email, got nil")
	}
}

func TestUpdateIdentity_NotFound(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@company.com"},
		},
	}

	newEmail := "new@company.com"
	updates := config.IdentityUpdates{Email: &newEmail}
	_, err := cfg.UpdateIdentity("nonexistent", updates)
	if err == nil {
		t.Fatal("expected error for nonexistent identity, got nil")
	}
}

func TestUpdateIdentity_GitName(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane Doe", Email: "work@company.com"},
		},
	}

	newGitName := "Jane Smith"
	updates := config.IdentityUpdates{GitName: &newGitName}
	identity, err := cfg.UpdateIdentity("work", updates)
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if identity.GitName != "Jane Smith" {
		t.Errorf("expected git name 'Jane Smith', got %q", identity.GitName)
	}
	if identity.Email != "work@company.com" {
		t.Errorf("expected email unchanged, got %q", identity.Email)
	}
}

func TestUpdateIdentity_HookMode(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@company.com"},
		},
	}

	mode := "block"
	updates := config.IdentityUpdates{HookMode: &mode}
	identity, err := cfg.UpdateIdentity("work", updates)
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if identity.HookMode != "block" {
		t.Errorf("expected hook mode 'block', got %q", identity.HookMode)
	}
}

func TestUpdateIdentity_InvalidHookMode(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@company.com"},
		},
	}

	mode := "invalid"
	updates := config.IdentityUpdates{HookMode: &mode}
	_, err := cfg.UpdateIdentity("work", updates)
	if err == nil {
		t.Fatal("expected error for invalid hook mode, got nil")
	}
}

func TestUpdateIdentity_ClearSSHKey(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@company.com", SSHKeyPath: "/home/user/.ssh/id_ed25519"},
		},
	}

	empty := ""
	updates := config.IdentityUpdates{SSHKeyPath: &empty}
	identity, err := cfg.UpdateIdentity("work", updates)
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if identity.SSHKeyPath != "" {
		t.Errorf("expected SSH key path cleared, got %q", identity.SSHKeyPath)
	}
}

func TestUpdateIdentity_CaseInsensitive(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "Work", Email: "work@company.com"},
		},
	}

	newEmail := "new@company.com"
	updates := config.IdentityUpdates{Email: &newEmail}
	identity, err := cfg.UpdateIdentity("work", updates)
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if identity.Name != "Work" {
		t.Errorf("expected original name 'Work' preserved, got %q", identity.Name)
	}
	if identity.Email != "new@company.com" {
		t.Errorf("expected email updated, got %q", identity.Email)
	}
}
