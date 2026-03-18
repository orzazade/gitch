package cmd

import (
	"testing"

	"github.com/orzazade/gitch/internal/config"
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
