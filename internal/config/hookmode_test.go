package config

import (
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/rules"
)

func Test_validateHookMode_Valid(t *testing.T) {
	for _, mode := range []string{"allow", "warn", "block", ""} {
		if err := validateHookMode(mode); err != nil {
			t.Errorf("validateHookMode(%q) returned error: %v", mode, err)
		}
	}
}

func Test_validateHookMode_Invalid(t *testing.T) {
	for _, mode := range []string{"invalid", "WARN", "Block", "none", "yes"} {
		err := validateHookMode(mode)
		if err == nil {
			t.Errorf("validateHookMode(%q) should return error", mode)
			continue
		}
		if !strings.Contains(err.Error(), "invalid hook mode") {
			t.Errorf("validateHookMode(%q) error = %v, want 'invalid hook mode'", mode, err)
		}
	}
}

func TestGetHookMode_Default(t *testing.T) {
	id := Identity{Name: "test", Email: "t@t.com"}
	if got := id.GetHookMode(); got != hookModeWarn {
		t.Errorf("GetHookMode() = %q, want %q", got, hookModeWarn)
	}
}

func TestGetHookMode_Explicit(t *testing.T) {
	for _, mode := range []string{hookModeAllow, hookModeWarn, hookModeBlock} {
		id := Identity{Name: "test", Email: "t@t.com", HookMode: mode}
		if got := id.GetHookMode(); got != mode {
			t.Errorf("GetHookMode() = %q, want %q", got, mode)
		}
	}
}

func TestUpdateIdentity_UpdateEmail(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "old@work.com"})
	newEmail := "new@work.com"

	updated, err := cfg.UpdateIdentity("work", IdentityUpdates{Email: &newEmail})
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if updated.Email != newEmail {
		t.Errorf("expected email %q, got %q", newEmail, updated.Email)
	}
}

func TestUpdateIdentity_UpdateGitName(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com", GitName: "Old Name"})
	newName := "New Name"

	updated, err := cfg.UpdateIdentity("work", IdentityUpdates{GitName: &newName})
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if updated.GitName != newName {
		t.Errorf("expected git name %q, got %q", newName, updated.GitName)
	}
}

func TestUpdateIdentity_NotFound(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com"})
	newEmail := "new@w.com"

	_, err := cfg.UpdateIdentity("nonexistent", IdentityUpdates{Email: &newEmail})
	if err == nil {
		t.Fatal("expected error for nonexistent identity")
	}
}

func TestUpdateIdentity_DuplicateEmail(t *testing.T) {
	cfg := testConfig(
		Identity{Name: "work", Email: "work@w.com"},
		Identity{Name: "personal", Email: "personal@p.com"},
	)
	dupEmail := "personal@p.com"

	_, err := cfg.UpdateIdentity("work", IdentityUpdates{Email: &dupEmail})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateIdentity_UpdateSSHKeyPath(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com"})
	sshPath := "/home/user/.ssh/work_key"

	updated, err := cfg.UpdateIdentity("work", IdentityUpdates{SSHKeyPath: &sshPath})
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if updated.SSHKeyPath != sshPath {
		t.Errorf("expected SSHKeyPath %q, got %q", sshPath, updated.SSHKeyPath)
	}
}

func TestUpdateIdentity_UpdateGPGKeyID(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com"})
	gpgKey := "ABCD1234"

	updated, err := cfg.UpdateIdentity("work", IdentityUpdates{GPGKeyID: &gpgKey})
	if err != nil {
		t.Fatalf("UpdateIdentity failed: %v", err)
	}
	if updated.GPGKeyID != gpgKey {
		t.Errorf("expected GPGKeyID %q, got %q", gpgKey, updated.GPGKeyID)
	}
}

func TestUpdateIdentity_InvalidEmail(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com"})
	bad := "not-an-email"

	_, err := cfg.UpdateIdentity("work", IdentityUpdates{Email: &bad})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestUpdateIdentity_InvalidHookMode(t *testing.T) {
	cfg := testConfig(Identity{Name: "work", Email: "w@w.com"})
	bad := "invalid"

	_, err := cfg.UpdateIdentity("work", IdentityUpdates{HookMode: &bad})
	if err == nil {
		t.Fatal("expected error for invalid hook mode")
	}
}

func TestAddRule_Success(t *testing.T) {
	cfg := testConfig()
	rule := rules.Rule{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"}

	if err := cfg.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}
}

func TestAddRule_DuplicatePattern(t *testing.T) {
	cfg := testConfig()
	rule := rules.Rule{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"}
	cfg.Rules = append(cfg.Rules, rule)

	err := cfg.AddRule(rule)
	if err == nil {
		t.Fatal("expected error for duplicate pattern")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddRule_InvalidPattern(t *testing.T) {
	cfg := testConfig()
	rule := rules.Rule{Type: rules.DirectoryRule, Pattern: "", Identity: "work"}

	err := cfg.AddRule(rule)
	if err == nil {
		t.Fatal("expected error for empty pattern")
	}
}

func TestRemoveRule_Success(t *testing.T) {
	cfg := testConfig()
	cfg.Rules = []rules.Rule{{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"}}

	if err := cfg.RemoveRule("/home/user/work"); err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}
	if len(cfg.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(cfg.Rules))
	}
}

func TestRemoveRule_NotFound(t *testing.T) {
	cfg := testConfig()

	err := cfg.RemoveRule("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent rule")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListRules_Empty(t *testing.T) {
	cfg := testConfig()
	if got := cfg.ListRules(); len(got) != 0 {
		t.Errorf("expected 0 rules, got %d", len(got))
	}
}

func TestListRules_WithRules(t *testing.T) {
	cfg := testConfig()
	cfg.Rules = []rules.Rule{
		{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"},
		{Type: rules.DirectoryRule, Pattern: "/home/user/personal", Identity: "personal"},
	}

	got := cfg.ListRules()
	if len(got) != 2 {
		t.Errorf("expected 2 rules, got %d", len(got))
	}
}
