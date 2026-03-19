package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/orzazade/gitch/internal/audit"
	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	statsLimit int
	statsAll   bool
	statsSince string
	statsJSON  bool
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show commit statistics grouped by identity",
	Long: `Show commit statistics for the current repository, grouped by author email
and mapped to gitch identities.

Helps you understand identity usage patterns and spot accidental wrong-identity
commits at a glance. If a rule matches the current directory or remote, the
expected identity is highlighted and unexpected emails are flagged.

Examples:
  gitch stats                 # Stats for last 1000 commits
  gitch stats --all           # Stats for entire history
  gitch stats --since 1.month # Stats for commits in the last month
  gitch stats --json          # Machine-readable output`,
	Args:          cobra.NoArgs,
	RunE:          runStats,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().IntVar(&statsLimit, "limit", 0, "Maximum commits to scan (default 1000, 0 for default)")
	statsCmd.Flags().BoolVar(&statsAll, "all", false, "Scan entire history (ignores --limit)")
	statsCmd.Flags().StringVar(&statsSince, "since", "", "Only count commits after date (e.g. 1.week, 2024-01-01)")
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output in JSON format")
}

type identityStats struct {
	Identity string  `json:"identity,omitempty"`
	Email    string  `json:"email"`
	Commits  int     `json:"commits"`
	Percent  float64 `json:"percent"`
	First    string  `json:"first_commit"`
	Last     string  `json:"last_commit"`
	Expected bool    `json:"expected,omitempty"`
	Unknown  bool    `json:"unknown,omitempty"`
}

type statsOutput struct {
	TotalCommits    int             `json:"total_commits"`
	ExpectedName    string          `json:"expected_name,omitempty"`
	ExpectedEmail   string          `json:"expected_email,omitempty"`
	RulePattern     string          `json:"rule_pattern,omitempty"`
	Authors         []identityStats `json:"authors"`
	UnexpectedCount int             `json:"unexpected_count"`
}

func runStats(cmd *cobra.Command, args []string) error {
	if !git.IsGitRepository() {
		return errNotGitRepo
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	commits, err := getStatsCommits()
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
	var expectedEmail string
	var expectedName string
	var rulePattern string
	cwd, err := os.Getwd()
	if err == nil {
		remoteURL, _ := rules.GetGitRemoteURL()
		if matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matched != nil {
			if identity, err := cfg.GetIdentity(matched.Identity); err == nil {
				expectedEmail = strings.ToLower(identity.Email)
				expectedName = identity.Name
				rulePattern = matched.Pattern
			}
		}
	}

	// Group commits by email
	type emailData struct {
		email string
		count int
		first string // oldest date
		last  string // newest date
	}
	byEmail := make(map[string]*emailData)
	for _, c := range commits {
		key := strings.ToLower(c.AuthorEmail)
		d := byEmail[key]
		if d == nil {
			d = &emailData{email: c.AuthorEmail}
			byEmail[key] = d
		}
		d.count++
		dateStr := c.Date.Format("2006-01-02")
		if d.first == "" || dateStr < d.first {
			d.first = dateStr
		}
		if dateStr > d.last {
			d.last = dateStr
		}
	}

	// Build sorted stats
	total := len(commits)
	var authors []identityStats
	var unexpectedCount int
	for key, d := range byEmail {
		s := identityStats{
			Email:   d.email,
			Commits: d.count,
			Percent: float64(d.count) / float64(total) * 100,
			First:   d.first,
			Last:    d.last,
		}
		if id, ok := identityByEmail[key]; ok {
			s.Identity = id.Name
		} else {
			s.Unknown = true
		}
		if expectedEmail != "" {
			if key == expectedEmail {
				s.Expected = true
			} else {
				unexpectedCount += d.count
			}
		}
		authors = append(authors, s)
	}

	// Sort: expected first, then by commit count descending
	sort.Slice(authors, func(i, j int) bool {
		if authors[i].Expected != authors[j].Expected {
			return authors[i].Expected
		}
		return authors[i].Commits > authors[j].Commits
	})

	out := statsOutput{
		TotalCommits:    total,
		ExpectedName:    expectedName,
		ExpectedEmail:   expectedEmail,
		RulePattern:     rulePattern,
		Authors:         authors,
		UnexpectedCount: unexpectedCount,
	}

	if statsJSON {
		return printJSON(out)
	}

	return printStatsHuman(out)
}

func getStatsCommits() ([]audit.Commit, error) {
	if statsSince != "" {
		return audit.GetCommitsSince(statsSince)
	}

	limit := statsLimit
	if statsAll {
		limit = 0 // unlimited
	} else if limit == 0 {
		limit = 1000
	}
	return audit.GetCommits(limit)
}

func printStatsHuman(out statsOutput) error {
	if out.ExpectedName != "" {
		fmt.Printf("Expected: %s (%s)",
			out.ExpectedName, out.ExpectedEmail)
		if out.RulePattern != "" {
			fmt.Printf("  %s", ui.DimStyle.Render("[rule: "+out.RulePattern+"]"))
		}
		fmt.Println()
		fmt.Println()
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  IDENTITY\tCOMMITS\t%\tFIRST\tLAST")

	for _, a := range out.Authors {
		marker := " "
		display := a.Email
		if a.Identity != "" {
			display = a.Identity
		}

		if a.Expected {
			marker = ui.SuccessStyle.Render("*")
			display = ui.SuccessStyle.Render(display)
		} else if out.ExpectedEmail != "" {
			// There is an expected identity but this isn't it
			marker = ui.WarningStyle.Render("!")
			display = ui.WarningStyle.Render(display)
		} else if a.Unknown {
			display = ui.DimStyle.Render(display)
		}

		pctStr := fmt.Sprintf("%.0f%%", a.Percent)
		fmt.Fprintf(w, "%s %s\t%d\t%s\t%s\t%s\n",
			marker, display, a.Commits, pctStr,
			ui.DimStyle.Render(a.First), ui.DimStyle.Render(a.Last))
	}
	w.Flush()

	fmt.Printf("\n%s %d commits scanned\n",
		ui.DimStyle.Render("Total:"), out.TotalCommits)

	if out.UnexpectedCount > 0 {
		fmt.Printf("\n%s %d commit(s) from unexpected identities.\n",
			ui.WarningStyle.Render("!"), out.UnexpectedCount)
	}

	return nil
}
