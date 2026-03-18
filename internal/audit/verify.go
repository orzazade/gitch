package audit

import (
	"fmt"
	"strings"

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
	resolved, err := resolveExpectedIdentity()
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, nil
	}

	localHashes := getLocalOnlyHashes()
	if localHashes == nil {
		return &VerifyResult{
			ExpectedEmail: resolved.identity.Email,
			MatchedRule:   resolved.rule,
			NoUpstream:    true,
		}, nil
	}

	if len(localHashes) == 0 {
		return &VerifyResult{
			ExpectedEmail: resolved.identity.Email,
			MatchedRule:   resolved.rule,
		}, nil
	}

	// Get recent commits (enough to cover all local-only ones)
	commits, err := GetCommits(len(localHashes) + 100)
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
		if !strings.EqualFold(commit.AuthorEmail, resolved.identity.Email) {
			mismatches = append(mismatches, Result{
				Commit:        commit,
				ExpectedEmail: resolved.identity.Email,
				IsMismatched:  true,
				IsPushed:      false,
			})
		}
	}

	return &VerifyResult{
		Mismatches:    mismatches,
		Total:         total,
		ExpectedEmail: resolved.identity.Email,
		MatchedRule:   resolved.rule,
	}, nil
}
