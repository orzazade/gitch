package rules

import (
	"path/filepath"
	"strings"
)

// Specificity calculates the specificity score for a rule
// Higher scores indicate more specific rules
// Directory rules: count path segments (*10), penalize wildcards (*-2)
// Remote rules: count parts (*10), exact repo match bonus (+50)
func (r Rule) Specificity() int {
	switch r.Type {
	case DirectoryRule:
		return directorySpecificity(r.Pattern)
	case RemoteRule:
		return remoteSpecificity(r.Pattern)
	default:
		return 0
	}
}

// directorySpecificity calculates specificity for directory patterns
func directorySpecificity(pattern string) int {
	expanded := expandAndAbs(pattern)

	// Count path segments
	segments := strings.Split(expanded, string(filepath.Separator))
	score := len(segments) * 10

	// Penalize wildcards
	wildcardCount := strings.Count(pattern, "*")
	score -= wildcardCount * 2

	// Double star is more general, penalize more
	doubleStarCount := strings.Count(pattern, "**")
	score -= doubleStarCount * 3

	return score
}

// remoteSpecificity calculates specificity for remote patterns
func remoteSpecificity(pattern string) int {
	// Count path parts
	parts := strings.Split(pattern, "/")
	score := len(parts) * 10

	wildcardCount := strings.Count(pattern, "*")
	if wildcardCount == 0 {
		// Exact repo match bonus
		score += 50
	} else {
		score -= wildcardCount * 2
	}

	return score
}

// Matches checks if a rule matches the given context
func (r Rule) Matches(cwd, remoteURL string) bool {
	switch r.Type {
	case DirectoryRule:
		matched, err := matchDirectory(r.Pattern, cwd)
		return err == nil && matched
	case RemoteRule:
		if remoteURL == "" {
			return false
		}
		parsed, err := parseRemote(remoteURL)
		if err != nil {
			return false
		}
		return matchRemote(r.Pattern, parsed)
	default:
		return false
	}
}

// FindBestMatch finds the rule with the highest specificity that matches the context
// Returns nil if no rules match
func FindBestMatch(rules []Rule, cwd, remoteURL string) *Rule {
	var bestMatch *Rule
	bestScore := -1

	for i := range rules {
		rule := &rules[i]
		if !rule.Matches(cwd, remoteURL) {
			continue
		}

		score := rule.Specificity()
		if score > bestScore {
			bestScore = score
			bestMatch = rule
		}
	}

	return bestMatch
}
