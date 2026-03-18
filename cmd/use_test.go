package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

func TestClosestIdentityName(t *testing.T) {
	identities := []config.Identity{
		{Name: "work"},
		{Name: "personal"},
		{Name: "oss"},
	}

	tests := []struct {
		input string
		want  string
	}{
		{"wrk", "work"},       // 1 deletion
		{"woork", "work"},     // 1 insertion
		{"prsonal", "personal"}, // 1 substitution
		{"os", "oss"},         // 1 insertion
		{"xyz123", ""},        // too distant — no suggestion
		{"work", "work"},      // exact match via distance 0
	}

	for _, tt := range tests {
		got := closestIdentityName(tt.input, identities)
		if got != tt.want {
			t.Errorf("closestIdentityName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRuleMismatchWarning(t *testing.T) {
	work := config.Identity{Name: "work", Email: "work@example.com"}
	personal := config.Identity{Name: "personal", Email: "personal@example.com"}

	cfgNoRules := &config.Config{Identities: []config.Identity{work}}
	if got := ruleMismatchWarning(&work, cfgNoRules); got != "" {
		t.Errorf("expected empty warning with no rules, got %q", got)
	}

	// Rule matches the identity being applied — no warning
	cfgMatchingRule := &config.Config{
		Identities: []config.Identity{work},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/nonexistent/path", Identity: "work"},
		},
	}
	if got := ruleMismatchWarning(&work, cfgMatchingRule); got != "" {
		t.Errorf("expected no warning when rule matches applied identity, got %q", got)
	}

	// Rule exists but matches a different identity — warning expected only when rule actually matches cwd
	// (can't fully test without real cwd match, but verify the no-match case)
	cfgOtherIdentity := &config.Config{
		Identities: []config.Identity{work, personal},
		Rules: []rules.Rule{
			{Type: rules.DirectoryRule, Pattern: "/nonexistent/path/that/never/matches", Identity: "personal"},
		},
	}
	// No match for cwd → no warning
	if got := ruleMismatchWarning(&work, cfgOtherIdentity); got != "" {
		t.Errorf("expected no warning when no rule matches cwd, got %q", got)
	}
}

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "ab", 1},
		{"ab", "abc", 1},
		{"abc", "xyz", 3},
		{"work", "wrk", 1},
		{"kitten", "sitting", 3},
	}
	for _, c := range cases {
		got := levenshtein(c.a, c.b)
		if got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
