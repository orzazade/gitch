package cmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var ruleSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest an identity rule for the current repository",
	Long: `Analyze the current repository's remote URL and your existing rules to
suggest which identity to use and what rule to create.

This is useful when you clone a new repository and want to quickly set up the
correct identity without manually figuring out which one to use.

The command looks at:
- The repository's remote URL (host and organization)
- Your existing rules for the same host or organization
- The current git identity (user.email) and whether it matches a gitch identity

Examples:
  cd ~/work/new-project && gitch rule suggest`,
	Args: cobra.NoArgs,
	RunE: runRuleSuggest,
}

func init() {
	ruleCmd.AddCommand(ruleSuggestCmd)
}

// ruleSuggestion holds a suggested rule with reasoning.
type ruleSuggestion struct {
	Identity string
	Pattern  string
	RuleType rules.RuleType
	Reason   string
}

// suggestRule analyzes the config, remote, and current identity to produce a suggestion.
func suggestRule(cfg *config.Config, remoteURL, currentEmail string) *ruleSuggestion {
	if remoteURL == "" {
		return nil
	}

	parsed, err := rules.ParseRemote(remoteURL)
	if err != nil || parsed == nil {
		return nil
	}

	// Strategy 1: Find existing rules for the same host/org and suggest the same identity.
	if suggestion := suggestFromExistingRules(cfg, parsed); suggestion != nil {
		return suggestion
	}

	// Strategy 2: Match current git email to a gitch identity.
	if suggestion := suggestFromCurrentEmail(cfg, parsed, currentEmail); suggestion != nil {
		return suggestion
	}

	return nil
}

// suggestFromExistingRules finds rules that share the same host/org and suggests
// using the same identity with a pattern covering this repo's org.
func suggestFromExistingRules(cfg *config.Config, remote *rules.ParsedRemote) *ruleSuggestion {
	type candidate struct {
		identity string
		matchOrg bool // true if the existing rule matches at the org level (stronger signal)
	}

	var candidates []candidate

	for _, rule := range cfg.Rules {
		if rule.Type != rules.RemoteRule {
			continue
		}

		ruleHost, rest, ok := strings.Cut(strings.ToLower(rule.Pattern), "/")
		if !ok {
			continue
		}

		ruleOrg, _, _ := strings.Cut(rest, "/")
		ruleOrg = strings.TrimSuffix(ruleOrg, "/*")

		if !strings.EqualFold(ruleHost, remote.Host) {
			continue
		}

		if remote.Org != "" && strings.EqualFold(ruleOrg, remote.Org) {
			// Same host AND same org — very strong signal
			candidates = append(candidates, candidate{identity: rule.Identity, matchOrg: true})
		} else {
			// Same host, different org — weaker signal
			candidates = append(candidates, candidate{identity: rule.Identity, matchOrg: false})
		}
	}

	// Prefer org-level matches
	for _, c := range candidates {
		if c.matchOrg {
			return &ruleSuggestion{
				Identity: c.identity,
				Pattern:  buildSuggestPattern(remote),
				RuleType: rules.RemoteRule,
				Reason:   "you already use '" + c.identity + "' for " + remote.Host + "/" + remote.Org + " repos",
			}
		}
	}

	// Fall back to host-level match
	if len(candidates) > 0 {
		c := candidates[0]
		return &ruleSuggestion{
			Identity: c.identity,
			Pattern:  buildSuggestPattern(remote),
			RuleType: rules.RemoteRule,
			Reason:   "you use '" + c.identity + "' for other repos on " + remote.Host,
		}
	}

	return nil
}

// suggestFromCurrentEmail checks if the current git email matches a gitch identity.
func suggestFromCurrentEmail(cfg *config.Config, remote *rules.ParsedRemote, currentEmail string) *ruleSuggestion {
	if currentEmail == "" {
		return nil
	}

	idx := slices.IndexFunc(cfg.Identities, func(id config.Identity) bool {
		return strings.EqualFold(id.Email, currentEmail)
	})
	if idx == -1 {
		return nil
	}

	return &ruleSuggestion{
		Identity: cfg.Identities[idx].Name,
		Pattern:  buildSuggestPattern(remote),
		RuleType: rules.RemoteRule,
		Reason:   "current git email (" + currentEmail + ") matches identity '" + cfg.Identities[idx].Name + "'",
	}
}

// buildSuggestPattern builds a remote rule pattern from a parsed remote.
// Returns "host/org/*" if org is set, or "host/*" otherwise.
func buildSuggestPattern(remote *rules.ParsedRemote) string {
	if remote.Org == "" {
		return remote.Host + "/*"
	}
	return remote.Host + "/" + remote.Org + "/*"
}

func runRuleSuggest(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errors.New("not a git repository; run this command from inside a repo")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Identities) == 0 {
		fmt.Println("No identities configured. Run 'gitch add' first.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	remoteURL, _ := rules.GetGitRemoteURL()

	// Check if a rule already matches
	existingMatch := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if existingMatch != nil {
		fmt.Printf("This repository already matches rule: %s -> %s\n",
			ui.NameStyle.Render(existingMatch.Pattern),
			ui.SuccessStyle.Render(existingMatch.Identity))
		fmt.Println(ui.DimStyle.Render("No suggestion needed."))
		return nil
	}

	if remoteURL == "" {
		fmt.Println("No git remote configured. Add a remote first, then try again.")
		return nil
	}

	_, currentEmail, _ := gitpkg.GetCurrentIdentity()

	suggestion := suggestRule(cfg, remoteURL, currentEmail)
	if suggestion == nil {
		fmt.Println("Could not determine which identity to suggest.")
		fmt.Println()
		fmt.Println("Available identities:")
		for _, id := range cfg.Identities {
			fmt.Printf("  %s (%s)\n", id.Name, id.Email)
		}
		fmt.Println()
		fmt.Println("Add a rule manually:")
		fmt.Println(ui.DimStyle.Render("  gitch rule add --remote \"host/org/*\" --use <identity>"))
		return nil
	}

	// Show the suggestion
	fmt.Printf("Remote: %s\n", remoteURL)
	fmt.Printf("Reason: %s\n", suggestion.Reason)
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("Suggested: use identity '" + suggestion.Identity + "'"))
	fmt.Println()
	fmt.Println("Run this to add the rule:")
	fmt.Printf("  gitch rule add --remote %q --use %s\n", suggestion.Pattern, suggestion.Identity)

	return nil
}
