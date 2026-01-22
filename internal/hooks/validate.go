package hooks

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
)

// ValidationResult contains the result of identity validation
type ValidationResult struct {
	Match            bool
	CurrentName      string
	CurrentEmail     string
	ExpectedName     string
	ExpectedEmail    string
	MatchedRule      *rules.Rule
	ExpectedIdentity *config.Identity
}

// Validate checks if current git identity matches expected for this context
func Validate() (*ValidationResult, error) {
	// 1. Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// 2. Get current git remote URL (may be empty)
	remoteURL, _ := rules.GetGitRemoteURL()

	// 3. Load config and find best matching rule
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	// 4. If no rule matches, validation passes (no expectation)
	if matchedRule == nil {
		return &ValidationResult{Match: true}, nil
	}

	// 5. Get expected identity from rule
	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return nil, fmt.Errorf("rule references unknown identity %q: %w", matchedRule.Identity, err)
	}

	// 6. Get current git identity
	currentName, currentEmail, err := git.GetCurrentIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to get current git identity: %w", err)
	}

	// 7. Compare (by email - more reliable than name)
	match := strings.EqualFold(currentEmail, expectedIdentity.Email)

	return &ValidationResult{
		Match:            match,
		CurrentName:      currentName,
		CurrentEmail:     currentEmail,
		ExpectedName:     expectedIdentity.Name,
		ExpectedEmail:    expectedIdentity.Email,
		MatchedRule:      matchedRule,
		ExpectedIdentity: expectedIdentity,
	}, nil
}

// FormatMismatch formats the validation result for display
func (r *ValidationResult) FormatMismatch() string {
	if r.MatchedRule == nil {
		return "Identity mismatch (no rule context)"
	}

	return fmt.Sprintf(
		"Identity mismatch!\n"+
			"  Expected: %s (%s)\n"+
			"  Current:  %s (%s)\n"+
			"  Rule:     %s -> %s",
		r.ExpectedName, r.ExpectedEmail,
		r.CurrentName, r.CurrentEmail,
		r.MatchedRule.Pattern, r.MatchedRule.Identity,
	)
}
