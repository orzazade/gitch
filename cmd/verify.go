package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/audit"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Check that unpushed commits match the expected identity",
	Long: `Verify that all local commits not yet pushed to the remote have the
correct author identity for this repository.

This is a quick pre-push safety check. It only examines unpushed commits,
making it fast even in large repositories. Use it before pushing, in a
pre-push hook, or in CI scripts.

Exit code 0 means all unpushed commits are correct (or there are none).
Exit code 1 means at least one commit has a mismatched identity.

Examples:
  gitch verify                   # Check unpushed commits
  gitch verify && git push       # Push only if identity is correct`,
	Args: cobra.NoArgs,
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepository() {
		return errors.New("not in a git repository")
	}

	result, err := audit.VerifyUnpushed()
	if err != nil {
		return fmt.Errorf("verify failed: %w", err)
	}

	if result == nil {
		fmt.Println("No identity rule matches this repository.")
		fmt.Println("Use 'gitch rule add' to create a rule for this directory or remote.")
		return nil
	}

	if result.NoUpstream {
		fmt.Println(ui.DimStyle.Render("No upstream branch configured — nothing to verify against."))
		fmt.Println("Push your branch first, then run 'gitch verify' to check future commits.")
		return nil
	}

	if result.Total == 0 {
		fmt.Println(ui.DimStyle.Render("No unpushed commits."))
		return nil
	}

	if len(result.Mismatches) == 0 {
		fmt.Printf("%s All %d unpushed commit(s) match %s.\n",
			ui.SuccessStyle.Render("OK"),
			result.Total,
			result.ExpectedEmail)
		return nil
	}

	// Print mismatches
	fmt.Printf("Expected: %s (rule: %s)\n\n",
		result.ExpectedEmail,
		result.MatchedRule.Pattern)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HASH\tAUTHOR\tDATE\tSUBJECT")
	for _, r := range result.Mismatches {
		subject := r.Commit.Subject
		if len(subject) > 50 {
			subject = subject[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			r.Commit.Hash[:8],
			ui.ErrorStyle.Render(r.Commit.AuthorEmail),
			r.Commit.Date.Format("2006-01-02"),
			subject)
	}
	w.Flush()

	fmt.Printf("\n%s %d of %d unpushed commit(s) have the wrong identity.\n",
		ui.ErrorStyle.Render("FAIL"),
		len(result.Mismatches),
		result.Total)
	fmt.Println("Fix with: gitch audit --fix")

	return errors.New("identity mismatch in unpushed commits")
}
