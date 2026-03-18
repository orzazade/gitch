package hooks

import (
	"strings"
	"testing"

	"github.com/orzazade/gitch/internal/rules"
)

func TestFormatMismatch_NoRuleContext(t *testing.T) {
	r := &ValidationResult{
		Match:         false,
		CurrentName:   "John",
		CurrentEmail:  "john@personal.com",
		ExpectedName:  "John",
		ExpectedEmail: "john@work.com",
		MatchedRule:   nil,
	}

	got := r.FormatMismatch()
	if got != "Identity mismatch (no rule context)" {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestFormatMismatch_WithRule(t *testing.T) {
	r := &ValidationResult{
		Match:         false,
		CurrentName:   "John Personal",
		CurrentEmail:  "john@personal.com",
		ExpectedName:  "John Work",
		ExpectedEmail: "john@work.com",
		MatchedRule: &rules.Rule{
			Pattern:  "/home/john/work",
			Identity: "work",
		},
	}

	got := r.FormatMismatch()

	if !strings.Contains(got, "Identity mismatch!") {
		t.Errorf("expected 'Identity mismatch!' header, got: %q", got)
	}
	if !strings.Contains(got, "john@work.com") {
		t.Errorf("expected expected email in output, got: %q", got)
	}
	if !strings.Contains(got, "john@personal.com") {
		t.Errorf("expected current email in output, got: %q", got)
	}
	if !strings.Contains(got, "/home/john/work") {
		t.Errorf("expected rule pattern in output, got: %q", got)
	}
	if !strings.Contains(got, "work") {
		t.Errorf("expected rule identity in output, got: %q", got)
	}
}

func TestFormatMismatch_ShowsBothIdentities(t *testing.T) {
	r := &ValidationResult{
		Match:         false,
		CurrentName:   "Alice",
		CurrentEmail:  "alice@oss.com",
		ExpectedName:  "Alice Corp",
		ExpectedEmail: "alice@corp.com",
		MatchedRule: &rules.Rule{
			Pattern:  "github.com/corp",
			Identity: "corp",
		},
	}

	got := r.FormatMismatch()

	// Both names should appear
	if !strings.Contains(got, "Alice Corp") {
		t.Errorf("expected name %q in output: %q", "Alice Corp", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("expected current name %q in output: %q", "Alice", got)
	}
	// Rule pattern and identity
	if !strings.Contains(got, "github.com/corp") {
		t.Errorf("expected rule pattern in output: %q", got)
	}
}
