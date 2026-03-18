package audit

import (
	"os/exec"
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/testutil"
)

func TestScan_NoRuleMatch(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	// Config with no rules matching this directory
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@e.com"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create a commit so there's something to scan
	for _, args := range [][]string{
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	result, err := Scan(ScanOptions{})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// No rule matches — empty result
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results when no rule matches, got %d", len(result.Results))
	}
}

func TestScan_WithMatchingRule(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create commits with wrong email
	for _, args := range [][]string{
		{"git", "config", "user.email", "wrong@email.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "wrong email commit"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	result, err := Scan(ScanOptions{})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if result.ExpectedEmail != "jane@work.com" {
		t.Errorf("expected email jane@work.com, got %q", result.ExpectedEmail)
	}
	if result.TotalScanned != 1 {
		t.Errorf("expected 1 scanned commit, got %d", result.TotalScanned)
	}
	if result.MismatchCount != 1 {
		t.Errorf("expected 1 mismatch, got %d", result.MismatchCount)
	}
}

func TestScan_ShowAllIncludesMatches(t *testing.T) {
	env := testutil.SetupGitEnv(t)
	defer env.Cleanup(t)
	env.Chdir(t)

	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", GitName: "Jane", Email: "jane@work.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: env.Dir, Identity: "work"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Create a commit with the correct email
	for _, args := range [][]string{
		{"git", "config", "user.email", "jane@work.com"},
		{"git", "config", "user.name", "Jane"},
		{"git", "commit", "--allow-empty", "-m", "correct email"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %v\n%s", args, err, out)
		}
	}

	result, err := Scan(ScanOptions{ShowAll: true})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	if result.MismatchCount != 0 {
		t.Errorf("expected 0 mismatches, got %d", result.MismatchCount)
	}
	// ShowAll should include matching commits
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result with ShowAll, got %d", len(result.Results))
	}
}
