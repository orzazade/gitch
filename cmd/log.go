package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/audit"
	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var logLimit int
var logJSON bool

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent commits with identity annotations",
	Long: `Show recent commits annotated with which gitch identity authored each one.

If a rule matches the current directory or remote, commits are checked against
the expected identity. Matching commits show the identity name in green,
wrong-identity commits are flagged in red, and commits from unknown emails
are shown in yellow.

This is a quick way to verify your recent work used the right identity before
pushing.

Examples:
  gitch log              # Show last 10 commits with identity info
  gitch log -n 20        # Show last 20 commits
  gitch log --json       # Output as JSON for scripting`,
	Args:          cobra.NoArgs,
	RunE:          runLog,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.Flags().IntVarP(&logLimit, "number", "n", 10, "Number of commits to show")
	logCmd.Flags().BoolVar(&logJSON, "json", false, "Output in JSON format")
}

type logEntry struct {
	Hash     string `json:"hash"`
	Date     string `json:"date"`
	Email    string `json:"email"`
	Subject  string `json:"subject"`
	Identity string `json:"identity"`
	Expected string `json:"expected,omitempty"`
	Match    string `json:"match"` // "ok", "mismatch", "unknown"
}

func runLog(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepository() {
		return errNotGitRepo
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if logLimit < 1 {
		logLimit = 10
	}

	commits, err := audit.GetCommits(logLimit)
	if err != nil {
		return fmt.Errorf("failed to get commits: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println(ui.DimStyle.Render("No commits found."))
		return nil
	}

	// Build email-to-identity lookup
	identityByEmail := make(map[string]*config.Identity)
	for i := range cfg.Identities {
		id := &cfg.Identities[i]
		identityByEmail[strings.ToLower(id.Email)] = id
	}

	// Find expected identity from rules
	var expectedIdentity *config.Identity
	var matchedRule *rules.Rule
	cwd, err := os.Getwd()
	if err == nil {
		remoteURL, _ := rules.GetGitRemoteURL()
		matchedRule = rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
		if matchedRule != nil {
			expectedIdentity, err = cfg.GetIdentity(matchedRule.Identity)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: rule '%s' references unknown identity '%s'\n", matchedRule.Pattern, matchedRule.Identity)
			}
		}
	}

	entries := make([]logEntry, 0, len(commits))
	for _, c := range commits {
		entry := logEntry{
			Hash:    shortHash(c.Hash),
			Date:    c.Date.Format("2006-01-02"),
			Email:   c.AuthorEmail,
			Subject: c.Subject,
		}

		entry.Subject = truncateSubject(entry.Subject, 50)

		// Match email to identity
		if id, ok := identityByEmail[strings.ToLower(c.AuthorEmail)]; ok {
			entry.Identity = id.Name
		}

		// Determine match status
		if expectedIdentity != nil {
			entry.Expected = expectedIdentity.Name
			if strings.EqualFold(c.AuthorEmail, expectedIdentity.Email) {
				entry.Match = "ok"
			} else {
				entry.Match = "mismatch"
			}
		} else if entry.Identity != "" {
			entry.Match = "ok"
		} else {
			entry.Match = "unknown"
		}

		entries = append(entries, entry)
	}

	if logJSON {
		return printJSON(entries)
	}

	// Print header with rule context
	if expectedIdentity != nil {
		fmt.Printf("Expected: %s (%s)",
			expectedIdentity.Name, expectedIdentity.Email)
		if matchedRule != nil {
			fmt.Printf("  %s", ui.DimStyle.Render("[rule: "+matchedRule.Pattern+"]"))
		}
		fmt.Println()
		fmt.Println()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  HASH\tDATE\tIDENTITY\tSUBJECT")
	for _, e := range entries {
		marker := " "
		identityDisplay := e.Identity
		if identityDisplay == "" {
			identityDisplay = ui.WarningStyle.Render("(" + e.Email + ")")
		}

		switch e.Match {
		case "ok":
			marker = ui.SuccessStyle.Render("*")
			if e.Identity != "" {
				identityDisplay = ui.SuccessStyle.Render(e.Identity)
			}
		case "mismatch":
			marker = ui.ErrorStyle.Render("!")
			if e.Identity != "" {
				identityDisplay = ui.ErrorStyle.Render(e.Identity)
			} else {
				identityDisplay = ui.ErrorStyle.Render("(" + e.Email + ")")
			}
		}

		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n",
			marker,
			ui.DimStyle.Render(e.Hash),
			ui.DimStyle.Render(e.Date),
			identityDisplay,
			e.Subject)
	}
	w.Flush()

	// Summary
	var mismatchCount int
	for _, e := range entries {
		if e.Match == "mismatch" {
			mismatchCount++
		}
	}
	if mismatchCount > 0 {
		fmt.Printf("\n%s %d commit(s) with wrong identity.\n",
			ui.ErrorStyle.Render("!"),
			mismatchCount)
	}

	return nil
}
