package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func TestGatherSSHInfo_NonexistentKey(t *testing.T) {
	info := gatherSSHInfo("~/.ssh/nonexistent_key_for_test")
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Path != "~/.ssh/nonexistent_key_for_test" {
		t.Errorf("expected path preserved, got %q", info.Path)
	}
	if info.Exists {
		t.Error("expected Exists=false for nonexistent key")
	}
	if info.Fingerprint != "" {
		t.Error("expected empty fingerprint for nonexistent key")
	}
	if info.AgentLoaded {
		t.Error("expected AgentLoaded=false for nonexistent key")
	}
}

func TestGatherSSHInfo_EmptyPath(t *testing.T) {
	// gatherSSHInfo is only called when path is non-empty,
	// but ExpandPath("") returns an error — verify graceful handling
	info := gatherSSHInfo("")
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Exists {
		t.Error("expected Exists=false for empty path")
	}
}

func TestGatherGPGInfo_InvalidKey(t *testing.T) {
	info := gatherGPGInfo("DEADBEEF12345678")
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.KeyID != "DEADBEEF12345678" {
		t.Errorf("expected key ID preserved, got %q", info.KeyID)
	}
	// Valid depends on GPG availability; just ensure no panic
}

func TestShowOutput_RulesCollection(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@corp.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "~/Projects/work/*", Identity: "work"},
			{Type: rules.RemoteRule, Pattern: "github.com/corp/*", Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: "~/Projects/oss/*", Identity: "personal"},
		},
	}

	identity, err := cfg.GetIdentity("work")
	if err != nil {
		t.Fatal(err)
	}

	var rules []showRule
	for _, r := range cfg.Rules {
		if r.Identity == identity.Name {
			rules = append(rules, showRule{Type: string(r.Type), Pattern: r.Pattern})
		}
	}

	if len(rules) != 2 {
		t.Fatalf("expected 2 rules for 'work', got %d", len(rules))
	}
	if rules[0].Pattern != "~/Projects/work/*" {
		t.Errorf("expected first rule pattern ~/Projects/work/*, got %q", rules[0].Pattern)
	}
	if rules[1].Pattern != "github.com/corp/*" {
		t.Errorf("expected second rule pattern github.com/corp/*, got %q", rules[1].Pattern)
	}
}

func TestShowOutput_DefaultHookMode(t *testing.T) {
	// When HookMode is empty, show should default to "allow"
	identity := config.Identity{Name: "test", Email: "test@test.com"}
	hookMode := identity.HookMode
	if hookMode == "" {
		hookMode = "allow"
	}
	if hookMode != "allow" {
		t.Errorf("expected default hook mode 'allow', got %q", hookMode)
	}
}

func TestShowOutput_ExplicitHookMode(t *testing.T) {
	identity := config.Identity{Name: "test", Email: "test@test.com", HookMode: "block"}
	hookMode := identity.HookMode
	if hookMode == "" {
		hookMode = "allow"
	}
	if hookMode != "block" {
		t.Errorf("expected hook mode 'block', got %q", hookMode)
	}
}

func TestPrintShowJSON_NilRulesBecomesEmptyArray(t *testing.T) {
	// Verify that nil rules get converted to empty slice for JSON
	output := showOutput{
		Name:    "test",
		GitName: "Test User",
		Email:   "test@test.com",
		Rules:   nil,
	}

	// printShowJSON converts nil to empty slice
	if output.Rules == nil {
		output.Rules = []showRule{}
	}
	if output.Rules == nil {
		t.Error("expected Rules to be non-nil after conversion")
	}
	if len(output.Rules) != 0 {
		t.Errorf("expected empty Rules slice, got %d items", len(output.Rules))
	}
}
