package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/audit"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	auditLimit   int
	auditAll     bool
	auditShowAll bool
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan repository for commits with mismatched identity",
	Long: `Scan git history for commits made with an identity that doesn't match
the expected identity for this repository.

The audit compares each commit's author email against the identity that
gitch's rules indicate should be used for this repository.

By default, scans the last 1000 commits. Use --limit to change this,
or --all to scan the entire history.

Examples:
  gitch audit                    # Scan last 1000 commits
  gitch audit --limit 100        # Scan last 100 commits
  gitch audit --all              # Scan entire history
  gitch audit --show-all         # Include matching commits in output`,
	Args: cobra.NoArgs,
	RunE: runAudit,
}

func init() {
	rootCmd.AddCommand(auditCmd)
	auditCmd.Flags().IntVar(&auditLimit, "limit", 0, "Maximum commits to scan (default 1000, 0 for default)")
	auditCmd.Flags().BoolVar(&auditAll, "all", false, "Scan entire history (ignores --limit)")
	auditCmd.Flags().BoolVar(&auditShowAll, "show-all", false, "Show all commits, not just mismatches")
}

func runAudit(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repo
	if !audit.IsGitRepo() {
		return fmt.Errorf("not in a git repository")
	}

	// Set limit based on flags
	limit := auditLimit
	if auditAll {
		limit = -1 // -1 means unlimited in Scan
	}

	// Run scan
	opts := audit.ScanOptions{
		Limit:   limit,
		ShowAll: auditShowAll,
	}
	result, err := audit.Scan(opts)
	if err != nil {
		return fmt.Errorf("audit failed: %w", err)
	}

	// Handle output
	return printAuditResults(result)
}

func printAuditResults(result *audit.ScanResult) error {
	// Handle no matching rule case
	if result.MatchedRule == nil {
		fmt.Println("No identity rule matches this repository.")
		fmt.Println("Use 'gitch rule add' to create a rule for this directory or remote.")
		return nil
	}

	// Print header with context
	fmt.Printf("Auditing against: %s (%s)\n",
		result.ExpectedEmail,
		result.MatchedRule.Pattern)
	fmt.Printf("Commits scanned: %d\n\n", result.TotalScanned)

	// Handle no mismatches
	if result.MismatchCount == 0 {
		fmt.Println(ui.SuccessStyle.Render("All commits match the expected identity."))
		return nil
	}

	// Print results table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tHASH\tAUTHOR\tDATE\tSUBJECT")

	for _, r := range result.Results {
		if !r.IsMismatched && !auditShowAll {
			continue
		}

		status := formatStatus(r)
		subject := truncateSubject(r.Commit.Subject, 50)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			status,
			r.Commit.Hash[:8],
			r.Commit.AuthorEmail,
			r.Commit.Date.Format("2006-01-02"),
			subject)
	}
	w.Flush()

	// Print summary
	fmt.Println()
	printSummary(result)

	return nil
}

func formatStatus(r audit.Result) string {
	if !r.IsMismatched {
		return ui.SuccessStyle.Render("OK")
	}
	if r.IsPushed {
		return ui.ErrorStyle.Render("PUSHED")
	}
	return ui.WarningStyle.Render("LOCAL")
}

func truncateSubject(subject string, maxLen int) string {
	if len(subject) <= maxLen {
		return subject
	}
	return subject[:maxLen-3] + "..."
}

func printSummary(result *audit.ScanResult) {
	if result.MismatchCount == 0 {
		return
	}

	fmt.Printf("Found %d mismatched commit(s):\n", result.MismatchCount)

	// Count local vs pushed mismatches from results
	var localMismatches, pushedMismatches int
	for _, r := range result.Results {
		if r.IsMismatched {
			if r.IsPushed {
				pushedMismatches++
			} else {
				localMismatches++
			}
		}
	}

	if localMismatches > 0 {
		msg := fmt.Sprintf("  %d local-only (safe to fix with 'gitch audit --fix')",
			localMismatches)
		fmt.Println(ui.WarningStyle.Render(msg))
	}

	if pushedMismatches > 0 {
		msg := fmt.Sprintf("  %d already pushed (requires force-push to fix)",
			pushedMismatches)
		fmt.Println(ui.ErrorStyle.Render(msg))
	}

	if result.NoUpstream {
		fmt.Println(ui.DimStyle.Render("  (No upstream branch - all commits shown as pushed)"))
	}
}
