package audit

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// VerifyResult contains the results of verifying unpushed commits.
type VerifyResult struct {
	// Mismatches contains commits that don't match the expected identity.
	Mismatches []Result
	// Total is the number of unpushed commits checked.
	Total int
	// ExpectedEmail is the email that commits should match.
	ExpectedEmail string
	// MatchedRule is the rule that determined the expected identity.
	MatchedRule *rules.Rule
	// NoUpstream is true if no upstream branch is configured.
	NoUpstream bool
}

// VerifyUnpushed checks that all unpushed commits match the expected identity
// for this repository. Returns nil VerifyResult with nil error if no rule matches.
func VerifyUnpushed() (*VerifyResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	remoteURL, _ := rules.GetGitRemoteURL()
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if matchedRule == nil {
		return nil, nil
	}

	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return nil, fmt.Errorf("rule references unknown identity %q: %w", matchedRule.Identity, err)
	}

	localHashes, _ := getLocalOnlyHashes()
	if localHashes == nil {
		return &VerifyResult{
			ExpectedEmail: expectedIdentity.Email,
			MatchedRule:   matchedRule,
			NoUpstream:    true,
		}, nil
	}

	if len(localHashes) == 0 {
		return &VerifyResult{
			ExpectedEmail: expectedIdentity.Email,
			MatchedRule:   matchedRule,
		}, nil
	}

	// Get recent commits (enough to cover all local-only ones)
	commits, err := getCommits(len(localHashes) + 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	var mismatches []Result
	var total int
	for _, commit := range commits {
		if !localHashes[commit.Hash] {
			continue
		}
		total++
		if !strings.EqualFold(commit.AuthorEmail, expectedIdentity.Email) {
			mismatches = append(mismatches, Result{
				Commit:        commit,
				ExpectedEmail: expectedIdentity.Email,
				IsMismatched:  true,
				IsPushed:      false,
			})
		}
	}

	return &VerifyResult{
		Mismatches:    mismatches,
		Total:         total,
		ExpectedEmail: expectedIdentity.Email,
		MatchedRule:   matchedRule,
	}, nil
}
