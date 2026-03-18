package audit

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
)

// Delimiters for parsing git log output
const (
	fieldDelim  = "|||"
	commitDelim = "<<<COMMIT>>>"
)

// defaultScanLimit is the number of commits scanned when no explicit limit is set.
const defaultScanLimit = 1000

// Commit represents a single git commit with metadata
type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Subject     string
}

// Result represents an audited commit with mismatch status
type Result struct {
	Commit        Commit
	ExpectedEmail string
	IsMismatched  bool
	IsPushed      bool // true = pushed to remote, false = local-only
}

// GetCommits retrieves commits from git log.
// If limit > 0, limits the number of commits returned.
// Returns empty slice with nil error for empty repos.
func GetCommits(limit int) ([]Commit, error) {
	// Build git log command with custom format
	// Format: <<<COMMIT>>>hash|||name|||email|||date|||subject
	formatArg := fmt.Sprintf("--format=%s%%H%s%%an%s%%ae%s%%ai%s%%s",
		commitDelim, fieldDelim, fieldDelim, fieldDelim, fieldDelim)

	args := []string{"log", formatArg}
	if limit > 0 {
		args = append(args, "--max-count="+strconv.Itoa(limit))
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check stderr for empty repo / no commits (git sends fatal messages to stderr)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "fatal: your current branch") ||
				strings.Contains(stderr, "does not have any commits") {
				return []Commit{}, nil
			}
		}
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	return parseCommits(string(output)), nil
}

// parseCommits parses the git log output into Commit structs.
// Malformed commit lines are silently skipped.
func parseCommits(output string) []Commit {
	output = strings.TrimSpace(output)
	if output == "" {
		return []Commit{}
	}

	// Split by commit delimiter
	parts := strings.Split(output, commitDelim)

	var commits []Commit
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		commit, err := parseCommitLine(part)
		if err != nil {
			// Skip malformed commits instead of failing entirely
			continue
		}
		commits = append(commits, commit)
	}

	return commits
}

// parseCommitLine parses a single commit line into a Commit struct
func parseCommitLine(line string) (Commit, error) {
	parts := strings.SplitN(line, fieldDelim, 5)
	if len(parts) < 5 {
		return Commit{}, fmt.Errorf("malformed commit line: expected 5 fields, got %d", len(parts))
	}

	// Parse the date
	dateStr := strings.TrimSpace(parts[3])
	date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		return Commit{}, fmt.Errorf("failed to parse date %q: %w", dateStr, err)
	}

	return Commit{
		Hash:        strings.TrimSpace(parts[0]),
		AuthorName:  strings.TrimSpace(parts[1]),
		AuthorEmail: strings.TrimSpace(parts[2]),
		Date:        date,
		Subject:     strings.TrimSpace(parts[4]),
	}, nil
}

// getLocalOnlyHashes returns a map of commit hashes that exist locally but not on the upstream.
// Returns nil if no upstream is configured (cannot determine pushed status).
// Returns empty map if all commits are pushed.
func getLocalOnlyHashes() map[string]bool {
	// Get upstream ref
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "@{u}")
	upstreamOutput, err := cmd.Output()
	if err != nil {
		// No upstream configured - can't determine pushed status
		return nil
	}

	upstream := strings.TrimSpace(string(upstreamOutput))
	if upstream == "" {
		return nil
	}

	// Get local-only commits (commits in HEAD but not in upstream)
	rangeArg := upstream + "..HEAD"
	cmd = exec.Command("git", "log", rangeArg, "--format=%H")
	output, err := cmd.Output()
	if err != nil {
		// If this fails, assume we can't determine status
		return nil
	}

	// Build map of local-only hashes
	localHashes := make(map[string]bool)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		hash := strings.TrimSpace(line)
		if hash != "" {
			localHashes[hash] = true
		}
	}

	return localHashes
}

// ScanOptions configures the Scan function behavior
type ScanOptions struct {
	Limit   int  // Max commits to scan (0 = default 1000)
	ShowAll bool // Include matching commits in results
}

// ScanResult contains the results of an audit scan
type ScanResult struct {
	Results        []Result
	ExpectedEmail  string
	MatchedRule    *rules.Rule
	TotalScanned   int
	MismatchCount  int
	LocalOnlyCount int
	PushedCount    int
	NoUpstream     bool // true if we couldn't determine pushed status
}

// matchedIdentity holds the resolved rule and identity for the current repository.
type matchedIdentity struct {
	rule     *rules.Rule
	identity *config.Identity
}

// resolveExpectedIdentity loads config, determines the current repo's matching
// rule, and resolves the expected identity. Returns (nil, nil) when no rule matches.
func resolveExpectedIdentity() (*matchedIdentity, error) {
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

	identity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		return nil, fmt.Errorf("rule references unknown identity %q: %w", matchedRule.Identity, err)
	}

	return &matchedIdentity{rule: matchedRule, identity: identity}, nil
}

// Scan performs an identity audit on the git history
// It compares commit author emails against the expected identity for this repo
func Scan(opts ScanOptions) (*ScanResult, error) {
	resolved, err := resolveExpectedIdentity()
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return &ScanResult{
			Results: []Result{},
		}, nil
	}

	// Handle limit:
	// - limit > 0: scan that many commits
	// - limit = 0: apply default (1000)
	// - limit < 0: scan all commits (unlimited)
	limit := opts.Limit
	if limit == 0 {
		limit = defaultScanLimit
	} else if limit < 0 {
		limit = 0 // Pass 0 to getCommits = unlimited (no --max-count flag)
	}

	// Get commits
	commits, err := GetCommits(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	// Get local-only hashes
	localHashes := getLocalOnlyHashes()
	noUpstream := localHashes == nil

	// Process commits
	var results []Result
	var mismatchCount, localOnlyCount, pushedCount int

	for _, commit := range commits {
		// Determine if pushed; if no upstream, assume pushed (conservative)
		isPushed := noUpstream || !localHashes[commit.Hash]

		// Count pushed vs local
		if isPushed {
			pushedCount++
		} else {
			localOnlyCount++
		}

		// Check for mismatch (case-insensitive email comparison)
		isMismatched := !strings.EqualFold(commit.AuthorEmail, resolved.identity.Email)
		if isMismatched {
			mismatchCount++
		}

		// Include in results if mismatch or ShowAll
		if isMismatched || opts.ShowAll {
			results = append(results, Result{
				Commit:        commit,
				ExpectedEmail: resolved.identity.Email,
				IsMismatched:  isMismatched,
				IsPushed:      isPushed,
			})
		}
	}

	return &ScanResult{
		Results:        results,
		ExpectedEmail:  resolved.identity.Email,
		MatchedRule:    resolved.rule,
		TotalScanned:   len(commits),
		MismatchCount:  mismatchCount,
		LocalOnlyCount: localOnlyCount,
		PushedCount:    pushedCount,
		NoUpstream:     noUpstream,
	}, nil
}
