package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func Test_testRulesForPath_BestMatchWins(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
			{Name: "company", Email: "company@example.com"},
			{Name: "personal", Email: "personal@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work/**", Identity: "work"},
			{Type: rules.DirectoryRule, Pattern: "/home/user/work/company/**", Identity: "company"},
			{Type: rules.DirectoryRule, Pattern: "/home/user/personal/**", Identity: "personal"},
		},
	}

	result := testRulesForPath(cfg, "/home/user/work/company/project", "")

	if len(result.Matches) != 2 {
		t.Fatalf("expected 2 matching rules, got %d", len(result.Matches))
	}

	if result.Best == nil {
		t.Fatal("expected a best match, got nil")
	}

	if result.Best.Rule.Identity != "company" {
		t.Errorf("expected best match identity %q, got %q", "company", result.Best.Rule.Identity)
	}

	if !result.Best.IsBest {
		t.Error("expected best match to have IsBest=true")
	}
}

func Test_testRulesForPath_NoMatchingRules(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "work", Email: "work@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work/**", Identity: "work"},
		},
	}

	result := testRulesForPath(cfg, "/opt/random/path", "")

	if len(result.Matches) != 0 {
		t.Fatalf("expected 0 matching rules, got %d", len(result.Matches))
	}

	if result.Best != nil {
		t.Error("expected no best match, got one")
	}
}

func Test_testRulesForPath_RemoteRuleMatches(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "oss", Email: "oss@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.RemoteRule, Pattern: "github.com/myorg/*", Identity: "oss"},
		},
	}

	result := testRulesForPath(cfg, "/tmp", "git@github.com:myorg/some-repo.git")

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 matching rule, got %d", len(result.Matches))
	}

	if result.Best == nil || result.Best.Rule.Identity != "oss" {
		t.Errorf("expected best match identity %q", "oss")
	}
}

func Test_testRulesForPath_MixedRulesSpecificityOrder(t *testing.T) {
	cfg := &config.Config{
		Identities: []config.Identity{
			{Name: "dir-work", Email: "dir@example.com"},
			{Name: "remote-work", Email: "remote@example.com"},
		},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work/**", Identity: "dir-work"},
			{Type: rules.RemoteRule, Pattern: "github.com/company/specific-repo", Identity: "remote-work"},
		},
	}

	// Exact remote (score 80) should beat directory wildcard
	result := testRulesForPath(cfg, "/home/user/work/project", "git@github.com:company/specific-repo.git")

	if len(result.Matches) != 2 {
		t.Fatalf("expected 2 matching rules, got %d", len(result.Matches))
	}

	if result.Best == nil {
		t.Fatal("expected a best match")
	}

	if result.Best.Rule.Identity != "remote-work" {
		t.Errorf("expected exact remote rule to win, got %q", result.Best.Rule.Identity)
	}
}
