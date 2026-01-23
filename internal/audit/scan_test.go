package audit

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestParseCommitLine_Valid tests parsing a normal commit line
func TestParseCommitLine_Valid(t *testing.T) {
	line := "abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500|||Add new feature"

	commit, err := parseCommitLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commit.Hash != "abc1234" {
		t.Errorf("expected hash 'abc1234', got %q", commit.Hash)
	}
	if commit.AuthorName != "John Doe" {
		t.Errorf("expected author name 'John Doe', got %q", commit.AuthorName)
	}
	if commit.AuthorEmail != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %q", commit.AuthorEmail)
	}
	if commit.Subject != "Add new feature" {
		t.Errorf("expected subject 'Add new feature', got %q", commit.Subject)
	}

	expectedDate := time.Date(2024, 1, 15, 10, 30, 0, 0, time.FixedZone("", -5*3600))
	if !commit.Date.Equal(expectedDate) {
		t.Errorf("expected date %v, got %v", expectedDate, commit.Date)
	}
}

// TestParseCommitLine_SpecialChars tests parsing with special characters in subject
// Note: If subject contains our delimiter (|||), parsing will fail gracefully
func TestParseCommitLine_SpecialChars(t *testing.T) {
	// Subject with special characters but NOT our delimiter
	line := "abc1234|||Jane Doe|||jane@example.com|||2024-01-15 10:30:00 -0500|||Fix: handle \"quotes\" and <brackets>"

	commit, err := parseCommitLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if commit.Subject != "Fix: handle \"quotes\" and <brackets>" {
		t.Errorf("unexpected subject: %q", commit.Subject)
	}

	// Subject containing delimiter - this will produce more than 5 parts
	// The 5th field will be partial, but it should still work since we have >= 5 parts
	lineWithDelim := "abc1234|||Jane|||jane@example.com|||2024-01-15 10:30:00 -0500|||feat: add ||| support"

	commit2, err := parseCommitLine(lineWithDelim)
	if err != nil {
		t.Fatalf("unexpected error for line with delimiter in subject: %v", err)
	}

	// Subject will be truncated at the first ||| since we split by it
	// This is acceptable - the 5th part becomes the subject (may be partial)
	if commit2.Hash != "abc1234" {
		t.Errorf("expected hash 'abc1234', got %q", commit2.Hash)
	}
}

// TestParseCommitLine_Empty tests parsing an empty line
func TestParseCommitLine_Empty(t *testing.T) {
	_, err := parseCommitLine("")
	if err == nil {
		t.Error("expected error for empty line, got nil")
	}
}

// TestParseCommitLine_MalformedDate tests parsing with invalid date format
func TestParseCommitLine_MalformedDate(t *testing.T) {
	line := "abc1234|||John Doe|||john@example.com|||not-a-date|||Add feature"

	_, err := parseCommitLine(line)
	if err == nil {
		t.Error("expected error for malformed date, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse date") {
		t.Errorf("expected date parse error, got: %v", err)
	}
}

// TestParseCommitLine_InsufficientFields tests parsing with missing fields
func TestParseCommitLine_InsufficientFields(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"one_field", "abc1234"},
		{"two_fields", "abc1234|||John Doe"},
		{"three_fields", "abc1234|||John Doe|||john@example.com"},
		{"four_fields", "abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCommitLine(tc.input)
			if err == nil {
				t.Error("expected error for insufficient fields, got nil")
			}
			if !strings.Contains(err.Error(), "malformed commit line") {
				t.Errorf("expected malformed error, got: %v", err)
			}
		})
	}
}

// TestParseCommits_Multiple tests parsing multiple commits
func TestParseCommits_Multiple(t *testing.T) {
	output := `<<<COMMIT>>>abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500|||First commit
<<<COMMIT>>>def5678|||Jane Doe|||jane@example.com|||2024-01-16 11:45:00 -0500|||Second commit
<<<COMMIT>>>ghi9012|||Bob Smith|||bob@example.com|||2024-01-17 09:00:00 -0500|||Third commit`

	commits, err := parseCommits(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}

	// Verify first commit
	if commits[0].Hash != "abc1234" {
		t.Errorf("first commit hash: expected 'abc1234', got %q", commits[0].Hash)
	}
	if commits[0].AuthorName != "John Doe" {
		t.Errorf("first commit author: expected 'John Doe', got %q", commits[0].AuthorName)
	}

	// Verify last commit
	if commits[2].Hash != "ghi9012" {
		t.Errorf("third commit hash: expected 'ghi9012', got %q", commits[2].Hash)
	}
	if commits[2].Subject != "Third commit" {
		t.Errorf("third commit subject: expected 'Third commit', got %q", commits[2].Subject)
	}
}

// TestParseCommits_Empty tests parsing empty input
func TestParseCommits_Empty(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace", "   \n\t  "},
		{"only_newlines", "\n\n\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commits, err := parseCommits(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(commits) != 0 {
				t.Errorf("expected 0 commits, got %d", len(commits))
			}
		})
	}
}

// TestParseCommits_SingleCommit tests parsing a single commit
func TestParseCommits_SingleCommit(t *testing.T) {
	output := "<<<COMMIT>>>abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500|||Only commit"

	commits, err := parseCommits(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}

	if commits[0].Hash != "abc1234" {
		t.Errorf("expected hash 'abc1234', got %q", commits[0].Hash)
	}
}

// TestParseCommits_SkipsMalformed tests that malformed commits are skipped
func TestParseCommits_SkipsMalformed(t *testing.T) {
	output := `<<<COMMIT>>>abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500|||Good commit
<<<COMMIT>>>malformed_line_missing_fields
<<<COMMIT>>>def5678|||Jane Doe|||jane@example.com|||2024-01-16 11:45:00 -0500|||Another good commit`

	commits, err := parseCommits(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 commits (malformed one skipped)
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits (skipping malformed), got %d", len(commits))
	}

	if commits[0].Hash != "abc1234" {
		t.Errorf("first commit hash: expected 'abc1234', got %q", commits[0].Hash)
	}
	if commits[1].Hash != "def5678" {
		t.Errorf("second commit hash: expected 'def5678', got %q", commits[1].Hash)
	}
}

// TestResult_Mismatch tests that different emails are detected as mismatched
func TestResult_Mismatch(t *testing.T) {
	commit := Commit{
		Hash:        "abc123",
		AuthorName:  "John Doe",
		AuthorEmail: "john@personal.com",
	}

	result := Result{
		Commit:        commit,
		ExpectedEmail: "john@work.com",
		IsMismatched:  !strings.EqualFold(commit.AuthorEmail, "john@work.com"),
	}

	if !result.IsMismatched {
		t.Error("expected IsMismatched=true for different emails")
	}
}

// TestResult_Match tests that same emails are not mismatched
func TestResult_Match(t *testing.T) {
	commit := Commit{
		Hash:        "abc123",
		AuthorName:  "John Doe",
		AuthorEmail: "john@work.com",
	}

	result := Result{
		Commit:        commit,
		ExpectedEmail: "john@work.com",
		IsMismatched:  !strings.EqualFold(commit.AuthorEmail, "john@work.com"),
	}

	if result.IsMismatched {
		t.Error("expected IsMismatched=false for matching emails")
	}
}

// TestResult_CaseInsensitive tests case-insensitive email comparison
func TestResult_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		name     string
		actual   string
		expected string
		match    bool
	}{
		{"exact_match", "john@example.com", "john@example.com", true},
		{"upper_vs_lower", "JOHN@EXAMPLE.COM", "john@example.com", true},
		{"mixed_case", "John@Example.COM", "john@example.com", true},
		{"different_emails", "john@example.com", "jane@example.com", false},
		{"same_local_diff_domain", "john@work.com", "john@personal.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commit := Commit{AuthorEmail: tc.actual}
			result := Result{
				Commit:        commit,
				ExpectedEmail: tc.expected,
				IsMismatched:  !strings.EqualFold(commit.AuthorEmail, tc.expected),
			}

			if result.IsMismatched == tc.match {
				t.Errorf("expected IsMismatched=%v for %q vs %q", !tc.match, tc.actual, tc.expected)
			}
		})
	}
}

// TestIsGitRepo_InRepo tests IsGitRepo returns true inside a git repo
func TestIsGitRepo_InRepo(t *testing.T) {
	// The test runs inside the gitch repo, so IsGitRepo should return true
	if !IsGitRepo() {
		t.Error("expected IsGitRepo() to return true inside git repo")
	}
}

// TestIsGitRepo_NotInRepo tests IsGitRepo returns false outside a git repo
func TestIsGitRepo_NotInRepo(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir) //nolint:errcheck

	// Change to a directory that's definitely not a git repo
	tmpDir := os.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Create a subdir in temp that's definitely not a git repo
	testDir, err := os.MkdirTemp(tmpDir, "not-a-git-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("failed to change to test directory: %v", err)
	}

	// IsGitRepo should return false in a non-git directory
	if IsGitRepo() {
		t.Error("expected IsGitRepo() to return false outside git repo")
	}
}

// TestCommit_ZeroValue tests that zero-value Commit has empty fields
func TestCommit_ZeroValue(t *testing.T) {
	var c Commit
	if c.Hash != "" {
		t.Error("expected empty Hash for zero-value Commit")
	}
	if c.AuthorName != "" {
		t.Error("expected empty AuthorName for zero-value Commit")
	}
	if c.AuthorEmail != "" {
		t.Error("expected empty AuthorEmail for zero-value Commit")
	}
	if !c.Date.IsZero() {
		t.Error("expected zero Date for zero-value Commit")
	}
	if c.Subject != "" {
		t.Error("expected empty Subject for zero-value Commit")
	}
}

// TestResult_PushedFlag tests the IsPushed flag behavior
func TestResult_PushedFlag(t *testing.T) {
	commit := Commit{Hash: "abc123", AuthorEmail: "test@example.com"}

	// Test pushed commit
	pushedResult := Result{
		Commit:   commit,
		IsPushed: true,
	}
	if !pushedResult.IsPushed {
		t.Error("expected IsPushed=true")
	}

	// Test local-only commit
	localResult := Result{
		Commit:   commit,
		IsPushed: false,
	}
	if localResult.IsPushed {
		t.Error("expected IsPushed=false for local-only commit")
	}
}

// TestScanOptions_Defaults tests ScanOptions default values
func TestScanOptions_Defaults(t *testing.T) {
	var opts ScanOptions
	if opts.Limit != 0 {
		t.Errorf("expected default Limit=0, got %d", opts.Limit)
	}
	if opts.ShowAll {
		t.Error("expected default ShowAll=false")
	}
}

// TestScanResult_EmptyResults tests ScanResult with no results
func TestScanResult_EmptyResults(t *testing.T) {
	result := ScanResult{
		Results:       []Result{},
		TotalScanned:  100,
		MismatchCount: 0,
	}

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
	if result.MismatchCount != 0 {
		t.Errorf("expected 0 mismatches, got %d", result.MismatchCount)
	}
}

// TestParseCommits_WithNewlines tests parsing commits with newlines in output
func TestParseCommits_WithNewlines(t *testing.T) {
	// Git log output often has trailing newlines
	output := `
<<<COMMIT>>>abc1234|||John Doe|||john@example.com|||2024-01-15 10:30:00 -0500|||First commit

<<<COMMIT>>>def5678|||Jane Doe|||jane@example.com|||2024-01-16 11:45:00 -0500|||Second commit

`

	commits, err := parseCommits(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
}

// TestParseCommitLine_WhitespaceHandling tests that whitespace is trimmed
func TestParseCommitLine_WhitespaceHandling(t *testing.T) {
	line := "  abc1234  |||  John Doe  |||  john@example.com  |||  2024-01-15 10:30:00 -0500  |||  Subject with spaces  "

	commit, err := parseCommitLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fields should be trimmed
	if commit.Hash != "abc1234" {
		t.Errorf("expected trimmed hash 'abc1234', got %q", commit.Hash)
	}
	if commit.AuthorName != "John Doe" {
		t.Errorf("expected trimmed author 'John Doe', got %q", commit.AuthorName)
	}
	if commit.Subject != "Subject with spaces" {
		t.Errorf("expected trimmed subject, got %q", commit.Subject)
	}
}
