package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	reposDepth int
	reposJSON  bool
)

var reposCmd = &cobra.Command{
	Use:   "repos <identity-name> [paths...]",
	Short: "List repositories that use a specific identity",
	Long: `Scan directories for git repositories and show only those that use
the given identity — either configured in local git config or expected
by a matching rule.

Useful before deleting or changing an identity to see which repos
would be affected.

By default, scans the current directory up to 3 levels deep.

Examples:
  gitch repos work
  gitch repos personal ~/Projects ~/OSS
  gitch repos work --depth 5
  gitch repos work --json`,
	Args:          cobra.MinimumNArgs(1),
	RunE:          runRepos,
	SilenceErrors: true,
	SilenceUsage:  true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeIdentityNames()
		}
		return nil, cobra.ShellCompDirectiveFilterDirs
	},
}

func init() {
	rootCmd.AddCommand(reposCmd)
	reposCmd.Flags().IntVar(&reposDepth, "depth", 3, "Maximum directory depth to search")
	reposCmd.Flags().BoolVar(&reposJSON, "json", false, "Output in JSON format")
}

type reposOutput struct {
	Identity string           `json:"identity"`
	Email    string           `json:"email"`
	Repos    []reposRepoEntry `json:"repos"`
	Total    int              `json:"total"`
}

type reposRepoEntry struct {
	Path      string `json:"path"`
	Match     string `json:"match"` // "active", "rule", or "both"
	HookLocal bool   `json:"hook_local"`
}

func runRepos(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := args[0]
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		return identityNotFoundError(name, cfg.ListIdentities(), "gitch repos "+name)
	}

	roots, err := resolveRoots(args[1:])
	if err != nil {
		return err
	}

	allRepos := findGitRepos(roots, reposDepth)
	if len(allRepos) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	results, err := scanRepos(allRepos, cfg)
	if err != nil {
		return err
	}

	// Filter to repos that match this identity
	var matched []reposRepoEntry
	for _, r := range results {
		activeMatch := r.Managed && strings.EqualFold(r.Identity, identity.Name)
		ruleMatch := strings.EqualFold(r.RuleMatch, identity.Name)

		if !activeMatch && !ruleMatch {
			continue
		}

		matchType := "active"
		if activeMatch && ruleMatch {
			matchType = "both"
		} else if ruleMatch {
			matchType = "rule"
		}

		matched = append(matched, reposRepoEntry{
			Path:      r.Path,
			Match:     matchType,
			HookLocal: r.HookLocal,
		})
	}

	if reposJSON {
		return printJSON(reposOutput{
			Identity: identity.Name,
			Email:    identity.Email,
			Repos:    matched,
			Total:    len(matched),
		})
	}

	printReposHuman(identity, matched, len(allRepos))
	return nil
}

func printReposHuman(identity *config.Identity, repos []reposRepoEntry, totalScanned int) {
	home, _ := os.UserHomeDir()

	fmt.Println(ui.DimStyle.Render("gitch repos — " + identity.Name + " (" + identity.Email + ")"))
	fmt.Println()

	if len(repos) == 0 {
		fmt.Println("  No repositories found using this identity.")
		fmt.Println(ui.DimStyle.Render("  Scanned " + strconv.Itoa(totalScanned) + " " + nounPlural(totalScanned, "repo", "repos") + "."))
		return
	}

	for _, r := range repos {
		displayPath := shortenHome(r.Path, home)

		var badge string
		switch r.Match {
		case "both":
			badge = ui.SuccessStyle.Render("active + rule")
		case "active":
			badge = ui.SuccessStyle.Render("active")
		case "rule":
			badge = ui.DimStyle.Render("rule only")
		}

		fmt.Printf("  %s  %s\n", badge, displayPath)
	}

	fmt.Println()
	fmt.Println(strconv.Itoa(len(repos)) + " of " + strconv.Itoa(totalScanned) + " " +
		nounPlural(totalScanned, "repo", "repos") + " " +
		nounPlural(len(repos), "uses", "use") + " " + identity.Name + ".")
}
