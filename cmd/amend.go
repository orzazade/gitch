package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/orzazade/gitch/internal/audit"
	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	amendJSON  bool
	amendForce bool
)

type amendOutput struct {
	OK           bool   `json:"ok"`
	Action       string `json:"action"`
	OldName      string `json:"old_name,omitempty"`
	OldEmail     string `json:"old_email,omitempty"`
	NewName      string `json:"new_name,omitempty"`
	NewEmail     string `json:"new_email,omitempty"`
	CommitHash   string `json:"commit_hash,omitempty"`
	IdentityName string `json:"identity_name,omitempty"`
	Reason       string `json:"reason"`
}

var amendCmd = &cobra.Command{
	Use:   "amend",
	Short: "Fix the last commit's author to match the expected identity",
	Long: `Amend the most recent commit so its author and committer match the
identity expected by the active rule for this directory or remote.

This is the quick fix when you accidentally commit with the wrong email.
It only rewrites the last commit — for bulk history rewriting, use gitch audit.

By default, amend refuses to rewrite commits that have already been pushed
to the remote. Use --force to override this safety check.

Examples:
  gitch amend                # Fix last commit's identity
  gitch amend --force        # Fix even if already pushed
  gitch amend --json         # Machine-readable output`,
	Args:          cobra.NoArgs,
	RunE:          runAmend,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(amendCmd)
	amendCmd.Flags().BoolVar(&amendJSON, "json", false, "Output in JSON format")
	amendCmd.Flags().BoolVar(&amendForce, "force", false, "Amend even if the commit has been pushed")
}

func runAmend(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	// Get the last commit's info
	commits, err := audit.GetCommits(1)
	if err != nil || len(commits) == 0 {
		return errors.New("no commits found")
	}
	lastCommit := commits[0]
	lastHash := lastCommit.Hash
	lastAuthorName := lastCommit.AuthorName
	lastAuthorEmail := lastCommit.AuthorEmail

	// Resolve expected identity from rules
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	remoteURL, _ := rules.GetGitRemoteURL()
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	if matchedRule == nil {
		out := amendOutput{
			OK:     false,
			Action: "none",
			Reason: "no rule matches this directory — cannot determine expected identity",
		}
		return printAmendResult(out)
	}

	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		out := amendOutput{
			OK:     false,
			Action: "none",
			Reason: "rule references unknown identity '" + matchedRule.Identity + "'",
		}
		return printAmendResult(out)
	}

	expectedName := expectedIdentity.GitAuthorName()
	expectedEmail := expectedIdentity.Email

	// Check if already correct
	if strings.EqualFold(lastAuthorEmail, expectedEmail) && lastAuthorName == expectedName {
		out := amendOutput{
			OK:           true,
			Action:       "none",
			OldName:      lastAuthorName,
			OldEmail:     lastAuthorEmail,
			CommitHash:   lastHash,
			IdentityName: expectedIdentity.Name,
			Reason:       "last commit already matches identity '" + expectedIdentity.Name + "'",
		}
		return printAmendResult(out)
	}

	// Safety check: refuse if commit is already pushed
	if !amendForce && isHeadPushed() {
		out := amendOutput{
			OK:       false,
			Action:   "blocked",
			OldName:  lastAuthorName,
			OldEmail: lastAuthorEmail,
			NewName:  expectedName,
			NewEmail: expectedEmail,
			Reason:   "commit is already pushed — use --force to rewrite",
		}
		return printAmendResult(out)
	}

	// Perform the amend
	author := expectedName + " <" + expectedEmail + ">"
	gitCmd := exec.Command("git", "commit", "--amend", "--no-edit", "--allow-empty", "--author="+author)
	gitCmd.Env = append(os.Environ(),
		"GIT_COMMITTER_NAME="+expectedName,
		"GIT_COMMITTER_EMAIL="+expectedEmail,
	)
	if output, err := gitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit --amend failed: %s", strings.TrimSpace(string(output)))
	}

	newCommits, _ := audit.GetCommits(1)
	newHash := ""
	if len(newCommits) > 0 {
		newHash = newCommits[0].Hash
	}

	out := amendOutput{
		OK:           true,
		Action:       "amended",
		OldName:      lastAuthorName,
		OldEmail:     lastAuthorEmail,
		NewName:      expectedName,
		NewEmail:     expectedEmail,
		CommitHash:   newHash,
		IdentityName: expectedIdentity.Name,
		Reason:       "commit amended to identity '" + expectedIdentity.Name + "'",
	}
	return printAmendResult(out)
}

// isHeadPushed returns true if HEAD has been pushed to the upstream branch.
func isHeadPushed() bool {
	// If @{u} doesn't exist, git errors out — treat as not pushed (safe to amend).
	// If it exists and output is empty, HEAD is already on the remote.
	out, err := exec.Command("git", "log", "--oneline", "@{u}..HEAD").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == ""
}

func printAmendResult(out amendOutput) error {
	if amendJSON {
		if err := printJSON(out); err != nil {
			return err
		}
		if !out.OK {
			return errors.New(out.Reason)
		}
		return nil
	}

	switch out.Action {
	case "amended":
		hash := shortHash(out.CommitHash)
		fmt.Printf("%s Amended commit %s\n",
			ui.SuccessStyle.Render("OK"),
			hash)
		fmt.Println("  Before: " + out.OldName + " (" + out.OldEmail + ")")
		fmt.Println("  After:  " + out.NewName + " (" + out.NewEmail + ")")
		return nil

	case "none":
		if out.OK {
			fmt.Printf("%s %s\n",
				ui.SuccessStyle.Render("OK"),
				out.Reason)
			return nil
		}
		fmt.Printf("%s %s\n", ui.ErrorStyle.Render("FAIL"), out.Reason)
		return errors.New(out.Reason)

	case "blocked":
		fmt.Printf("%s %s\n",
			ui.ErrorStyle.Render("FAIL"),
			"Commit is already pushed to the remote.")
		fmt.Println("  Current: " + out.OldName + " (" + out.OldEmail + ")")
		fmt.Println("  Wanted:  " + out.NewName + " (" + out.NewEmail + ")")
		fmt.Println()
		fmt.Println("Use --force to rewrite the pushed commit.")
		return errors.New(out.Reason)
	}

	return nil
}
