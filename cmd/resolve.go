package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var resolveJSON bool

type resolveOutput struct {
	Query    string `json:"query"`
	Type     string `json:"type"`
	Identity string `json:"identity,omitempty"`
	Email    string `json:"email,omitempty"`
	GitName  string `json:"git_name,omitempty"`
	RuleType string `json:"rule_type,omitempty"`
	Pattern  string `json:"rule_pattern,omitempty"`
	Matched  bool   `json:"matched"`
}

var resolveCmd = &cobra.Command{
	Use:   "resolve <path-or-url>",
	Short: "Show which identity would be used for a path or remote URL",
	Long: `Preview which identity gitch would apply for a given directory path or
remote URL, without needing to be in that directory or clone the repo.

Useful for verifying rules before cloning or entering a directory:

  gitch resolve ~/work/new-project
  gitch resolve git@github.com:company/repo.git
  gitch resolve github.com/personal/dotfiles

The argument is treated as a remote URL if it contains ":" (SCP-style),
"://" (HTTPS/SSH), or matches a host/path pattern with no path separator
prefix. Otherwise it is treated as a directory path.

Exit code 0 if a matching rule is found, 1 if no rule matches.

Examples:
  gitch resolve ~/work/new-project
  gitch resolve /home/user/oss/contrib
  gitch resolve git@github.com:company/repo.git
  gitch resolve https://github.com/personal/dotfiles.git
  gitch resolve github.com/myorg/myrepo
  gitch resolve --json ~/work/something`,
	Args:          cobra.ExactArgs(1),
	RunE:          runResolve,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().BoolVar(&resolveJSON, "json", false, "Output in JSON format")
}

func runResolve(cmd *cobra.Command, args []string) error {
	query := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Rules) == 0 {
		if resolveJSON {
			return printJSON(resolveOutput{Query: query, Matched: false})
		}
		fmt.Println("No rules configured. Add rules with 'gitch rule add'.")
		return errors.New("no rules configured")
	}

	queryType, cwd, remoteURL := classifyQuery(query)

	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	out := resolveOutput{
		Query:   query,
		Type:    queryType,
		Matched: matchedRule != nil,
	}

	if matchedRule != nil {
		out.RuleType = string(matchedRule.Type)
		out.Pattern = matchedRule.Pattern
		out.Identity = matchedRule.Identity

		if identity, err := cfg.GetIdentity(matchedRule.Identity); err == nil {
			out.Email = identity.Email
			out.GitName = identity.GitAuthorName()
		}
	}

	if resolveJSON {
		return printJSON(out)
	}

	return printResolveHuman(out)
}

// classifyQuery determines whether the input is a directory path or remote URL,
// and returns the appropriate (cwd, remoteURL) pair for rule matching.
func classifyQuery(query string) (queryType, cwd, remoteURL string) {
	// SCP-style: git@host:path
	if strings.Contains(query, "://") || isSCPStyle(query) {
		return "remote", "", query
	}

	// Bare host/path like "github.com/org/repo" — no ~ or / prefix, contains a dot before first /
	if !strings.HasPrefix(query, "/") && !strings.HasPrefix(query, "~") && !strings.HasPrefix(query, ".") {
		if host, _, ok := strings.Cut(query, "/"); ok && strings.Contains(host, ".") {
			// Normalize to a proper URL so ParseRemote can handle it.
			return "remote", "", "https://" + query
		}
	}

	// Directory path — expand ~ and resolve to absolute
	expanded := query
	if e, err := sshpkg.ExpandPath(query); err == nil {
		expanded = e
	}
	if abs, err := filepath.Abs(expanded); err == nil {
		expanded = abs
	}
	return "directory", expanded, ""
}

// isSCPStyle returns true for SCP-style git URLs like git@host:path.
func isSCPStyle(s string) bool {
	// Must have : but not be a Windows path (C:\...) and not have :// (URL scheme)
	colonIdx := strings.Index(s, ":")
	if colonIdx < 0 {
		return false
	}
	if strings.Contains(s, "://") {
		return false
	}
	// Must have @ before : (git@host:path) or be user@host:path
	return strings.Contains(s[:colonIdx], "@")
}

func printResolveHuman(out resolveOutput) error {
	fmt.Printf("Query:    %s (%s)\n", out.Query, out.Type)

	if !out.Matched {
		fmt.Println(ui.WarningStyle.Render("No rule matches this " + out.Type + "."))
		return errors.New("no matching rule")
	}

	fmt.Printf("Rule:     %s '%s'\n", out.RuleType, out.Pattern)
	if out.Identity != "" {
		label := out.Identity
		if out.Email != "" {
			label += " (" + out.Email + ")"
		}
		fmt.Printf("Identity: %s\n", ui.SuccessStyle.Render(label))
		if out.GitName != "" {
			fmt.Printf("Author:   %s\n", out.GitName)
		}
	}
	return nil
}
