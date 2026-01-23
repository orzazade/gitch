package audit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/orzazade/gitch/internal/ui"
)

// ConfirmPhrase is the exact phrase users must type to confirm destructive operations.
const ConfirmPhrase = "I UNDERSTAND"

// GenerateMailmap creates mailmap content to remap wrong emails to the expected email.
// Mailmap format: <correct-email> <wrong-email>
func GenerateMailmap(mismatches []Result, expectedEmail string) string {
	// Collect unique wrong emails
	uniqueEmails := make(map[string]bool)
	for _, r := range mismatches {
		if r.IsMismatched {
			uniqueEmails[r.Commit.AuthorEmail] = true
		}
	}

	// Generate mailmap lines
	var lines []string
	for wrongEmail := range uniqueEmails {
		lines = append(lines, fmt.Sprintf("<%s> <%s>", expectedEmail, wrongEmail))
	}

	return strings.Join(lines, "\n")
}

// RunFilterRepo executes git-filter-repo with the given mailmap file.
// Uses --force to override fresh clone check (we have backup).
// Pipes stdout/stderr for progress visibility.
func RunFilterRepo(mailmapPath string) error {
	cmd := exec.Command("git", "filter-repo", "--force", "--mailmap", mailmapPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git-filter-repo failed: %w", err)
	}

	return nil
}

// GetRemotes returns a list of remote names configured in the repository.
func GetRemotes() ([]string, error) {
	cmd := exec.Command("git", "remote")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	// Split output by newlines and filter empty strings
	var remotes []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			remotes = append(remotes, line)
		}
	}

	return remotes, nil
}

// RemoveRemotes removes all configured remotes from the repository.
// This prevents accidental force-push after history rewrite.
// Ignores "remote does not exist" errors.
func RemoveRemotes() error {
	remotes, err := GetRemotes()
	if err != nil {
		return err
	}

	for _, remote := range remotes {
		cmd := exec.Command("git", "remote", "remove", remote)
		if err := cmd.Run(); err != nil {
			// Ignore "remote does not exist" errors (exit code 2)
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 2 {
					continue
				}
			}
			return fmt.Errorf("failed to remove remote %s: %w", remote, err)
		}
	}

	return nil
}

// Fix rewrites git history to correct mismatched commit identities.
// This is a destructive operation with multiple safety guardrails:
// 1. Checks git-filter-repo availability
// 2. Creates mirror backup before any changes
// 3. Shows GPG signature loss warning
// 4. Requires typed confirmation ("I UNDERSTAND")
// 5. Removes remotes after rewrite to prevent accidental force-push
func Fix(scanResult *ScanResult) error {
	// Step 1: Prerequisites check
	if !IsFilterRepoAvailable() {
		return fmt.Errorf("git-filter-repo not found\n\nInstall with:\n  brew install git-filter-repo\n  # or: pip install git-filter-repo")
	}

	// Step 2: Collect commits that need fixing
	var toFix []Result
	for _, r := range scanResult.Results {
		if r.IsMismatched {
			toFix = append(toFix, r)
		}
	}

	if len(toFix) == 0 {
		return fmt.Errorf("no mismatched commits to fix")
	}

	// Count pushed vs local among mismatches
	var localCount, pushedCount int
	for _, r := range toFix {
		if r.IsPushed {
			pushedCount++
		} else {
			localCount++
		}
	}

	// Step 3: Show what will happen
	fmt.Printf("Will rewrite %d commit(s):\n", len(toFix))
	fmt.Printf("  - %d local-only (safe)\n", localCount)
	if pushedCount > 0 {
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("  - %d already pushed (will require force-push)", pushedCount)))
	}

	// Step 4: GPG warning (AUDIT-07)
	fmt.Println()
	fmt.Println(ui.ErrorStyle.Render("WARNING: GPG signatures will be PERMANENTLY LOST for all rewritten commits."))
	fmt.Println("This cannot be undone. Re-signing would create different commit hashes.")

	// Step 5: Typed confirmation (AUDIT-08)
	confirmed, err := ui.TypedConfirm("\nThis operation rewrites git history and cannot be undone.", ConfirmPhrase)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Step 6: Create backup (AUDIT-05)
	// Get repo name for backup path
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}
	repoRoot := strings.TrimSpace(string(output))
	repoName := filepath.Base(repoRoot)

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s-backup-%s", repoName, timestamp))

	fmt.Printf("\nCreating backup at: %s\n", backupPath)
	if err := CreateMirrorBackup(backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// Step 7: Generate and write mailmap
	mailmapContent := GenerateMailmap(toFix, scanResult.ExpectedEmail)
	mailmapPath := filepath.Join(os.TempDir(), "gitch-mailmap")
	if err := os.WriteFile(mailmapPath, []byte(mailmapContent), 0644); err != nil {
		return fmt.Errorf("failed to write mailmap: %w", err)
	}
	defer os.Remove(mailmapPath)

	// Step 8: Run git-filter-repo (AUDIT-04)
	fmt.Println("\nRewriting history...")
	if err := RunFilterRepo(mailmapPath); err != nil {
		return fmt.Errorf("git-filter-repo failed: %w\n\nYour backup is at: %s", err, backupPath)
	}

	// Step 9: Remove remotes (AUDIT-06)
	remotesBefore, _ := GetRemotes()
	if err := RemoveRemotes(); err != nil {
		// Non-fatal: warn but continue
		fmt.Println(ui.WarningStyle.Render(fmt.Sprintf("\nWarning: failed to remove remotes: %v", err)))
	}
	if len(remotesBefore) > 0 {
		fmt.Println(ui.WarningStyle.Render("\nRemote(s) removed to prevent accidental force-push."))
		fmt.Println("When ready to push rewritten history:")
		fmt.Println("  git remote add origin <url>")
		fmt.Println("  git push --force-with-lease")
	}

	// Step 10: Success message
	fmt.Println(ui.SuccessStyle.Render("\nHistory rewritten successfully."))
	fmt.Printf("Backup preserved at: %s\n", backupPath)

	return nil
}
