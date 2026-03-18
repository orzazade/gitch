package config

import (
	"testing"

	"github.com/orzazade/gitch/internal/rules"
)

func TestIsDirectoryOverlap(t *testing.T) {
	cases := []struct {
		p1, p2 string
		want   bool
	}{
		// Exact match
		{"/home/user/work", "/home/user/work", true},
		// Parent contains child
		{"/home/user", "/home/user/work", true},
		{"/home/user/work", "/home/user", true},
		// Wildcard variants treated as their base
		{"/home/user/work/*", "/home/user/work/projects", true},
		{"/home/user/work/**", "/home/user/work", true},
		// Completely different directories
		{"/home/user/work", "/home/user/personal", false},
		// Sibling dirs (no overlap)
		{"/home/user/work", "/home/user/workshop", true}, // current prefix-match behavior
		// Empty patterns
		{"", "", true},
	}

	for _, c := range cases {
		got := isDirectoryOverlap(c.p1, c.p2)
		if got != c.want {
			t.Errorf("isDirectoryOverlap(%q, %q) = %v, want %v", c.p1, c.p2, got, c.want)
		}
	}
}

func TestIsRemoteOverlap(t *testing.T) {
	cases := []struct {
		p1, p2 string
		want   bool
	}{
		// Different hosts — no overlap
		{"github.com/org/repo", "gitlab.com/org/repo", false},
		// Same host, same org — overlap
		{"github.com/myorg", "github.com/myorg/repo", true},
		// Same host, different orgs — no overlap
		{"github.com/orgA/repo", "github.com/orgB/repo", false},
		// Same host, same org wildcard
		{"github.com/myorg/*", "github.com/myorg/repo", true},
		// Host only vs host+path — overlap (host covers everything)
		{"github.com", "github.com/any/repo", true},
		// Exact match
		{"github.com/org/repo", "github.com/org/repo", true},
	}

	for _, c := range cases {
		got := isRemoteOverlap(c.p1, c.p2)
		if got != c.want {
			t.Errorf("isRemoteOverlap(%q, %q) = %v, want %v", c.p1, c.p2, got, c.want)
		}
	}
}

func TestFindOverlappingRules_NoOverlap(t *testing.T) {
	cfg := &Config{
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/personal", Identity: "personal"},
		},
	}

	newRule := rules.Rule{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"}
	overlapping := cfg.FindOverlappingRules(newRule)
	if len(overlapping) != 0 {
		t.Errorf("expected 0 overlapping rules, got %d", len(overlapping))
	}
}

func TestFindOverlappingRules_WithOverlap(t *testing.T) {
	cfg := &Config{
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"},
		},
	}

	// Child of existing rule — should overlap
	newRule := rules.Rule{Type: rules.DirectoryRule, Pattern: "/home/user/work/projectA", Identity: "work2"}
	overlapping := cfg.FindOverlappingRules(newRule)
	if len(overlapping) != 1 {
		t.Errorf("expected 1 overlapping rule, got %d", len(overlapping))
	}
}

func TestFindOverlappingRules_SkipsExactDuplicate(t *testing.T) {
	cfg := &Config{
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"},
		},
	}

	// Exact same pattern — FindOverlappingRules skips it (AddRule handles duplicates separately)
	newRule := rules.Rule{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"}
	overlapping := cfg.FindOverlappingRules(newRule)
	if len(overlapping) != 0 {
		t.Errorf("expected exact duplicate to be skipped, got %d overlapping", len(overlapping))
	}
}

func TestFindOverlappingRules_DifferentTypes(t *testing.T) {
	cfg := &Config{
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/home/user/work", Identity: "work"},
		},
	}

	// Remote rule — different type, should not conflict with directory rule
	newRule := rules.Rule{Type: rules.RemoteRule, Pattern: "github.com/org", Identity: "work"}
	overlapping := cfg.FindOverlappingRules(newRule)
	if len(overlapping) != 0 {
		t.Errorf("expected 0 overlapping rules across different types, got %d", len(overlapping))
	}
}
