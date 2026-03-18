package cmd

import (
	"fmt"
	"os"
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
func suggestRule(cfg *config.Config, cwd, remoteURL, currentEmail string) *ruleSuggestion {
	if remoteURL == "" {
		return nil
	}

	parsed, err := parseRemoteForSuggest(remoteURL)
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
func suggestFromExistingRules(cfg *config.Config, remote *suggestParsedRemote) *ruleSuggestion {
	type candidate struct {
		identity string
		matchOrg bool // true if the existing rule matches at the org level (stronger signal)
	}

	var candidates []candidate

	for _, rule := range cfg.Rules {
		if rule.Type != rules.RemoteRule {
			continue
		}

		parts := strings.SplitN(strings.ToLower(rule.Pattern), "/", 3)
		if len(parts) < 2 {
			continue
		}

		ruleHost := parts[0]
		ruleOrg := strings.TrimSuffix(parts[1], "/*")

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
			pattern := remote.Host + "/" + remote.Org + "/*"
			return &ruleSuggestion{
				Identity: c.identity,
				Pattern:  pattern,
				RuleType: rules.RemoteRule,
				Reason:   fmt.Sprintf("you already use '%s' for %s/%s repos", c.identity, remote.Host, remote.Org),
			}
		}
	}

	// Fall back to host-level match
	if len(candidates) > 0 {
		c := candidates[0]
		pattern := remote.Host + "/" + remote.Org + "/*"
		if remote.Org == "" {
			pattern = remote.Host + "/*"
		}
		return &ruleSuggestion{
			Identity: c.identity,
			Pattern:  pattern,
			RuleType: rules.RemoteRule,
			Reason:   fmt.Sprintf("you use '%s' for other repos on %s", c.identity, remote.Host),
		}
	}

	return nil
}

// suggestFromCurrentEmail checks if the current git email matches a gitch identity.
func suggestFromCurrentEmail(cfg *config.Config, remote *suggestParsedRemote, currentEmail string) *ruleSuggestion {
	if currentEmail == "" {
		return nil
	}

	for _, id := range cfg.Identities {
		if !strings.EqualFold(id.Email, currentEmail) {
			continue
		}

		pattern := remote.Host + "/" + remote.Org + "/*"
		if remote.Org == "" {
			pattern = remote.Host + "/*"
		}
		return &ruleSuggestion{
			Identity: id.Name,
			Pattern:  pattern,
			RuleType: rules.RemoteRule,
			Reason:   fmt.Sprintf("current git email (%s) matches identity '%s'", currentEmail, id.Name),
		}
	}

	return nil
}

// suggestParsedRemote is a simple host/org/repo struct for suggestion logic.
type suggestParsedRemote struct {
	Host string
	Org  string
	Repo string
}

// parseRemoteForSuggest extracts host/org/repo from a remote URL for suggestions.
// Uses the same git-urls parser as the rules package.
func parseRemoteForSuggest(rawURL string) (*suggestParsedRemote, error) {
	// Get the remote URL and strip common prefixes/suffixes to extract host/org/repo.
	// Delegate to rules package's GetGitRemoteURL for fetching; here we just parse.
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return nil, fmt.Errorf("empty remote URL")
	}

	// Handle SSH format: git@host:org/repo.git
	if strings.Contains(url, "@") && strings.Contains(url, ":") && !strings.Contains(url, "://") {
		// SCP-style: git@github.com:org/repo.git
		atIdx := strings.Index(url, "@")
		colonIdx := strings.Index(url[atIdx:], ":") + atIdx
		host := strings.ToLower(url[atIdx+1 : colonIdx])
		path := strings.TrimSuffix(url[colonIdx+1:], ".git")
		parts := strings.SplitN(path, "/", 2)
		result := &suggestParsedRemote{Host: host}
		if len(parts) >= 1 {
			result.Org = parts[0]
		}
		if len(parts) >= 2 {
			result.Repo = parts[1]
		}
		return result, nil
	}

	// Handle HTTPS format: https://host/org/repo.git
	stripped := url
	for _, prefix := range []string{"https://", "http://", "ssh://", "git://"} {
		stripped = strings.TrimPrefix(stripped, prefix)
	}
	stripped = strings.TrimSuffix(stripped, ".git")

	parts := strings.SplitN(stripped, "/", 3)
	result := &suggestParsedRemote{}
	if len(parts) >= 1 {
		result.Host = strings.ToLower(parts[0])
	}
	if len(parts) >= 2 {
		result.Org = parts[1]
	}
	if len(parts) >= 3 {
		result.Repo = parts[2]
	}

	if result.Host == "" {
		return nil, fmt.Errorf("could not parse host from remote URL: %s", rawURL)
	}

	return result, nil
}

func runRuleSuggest(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return fmt.Errorf("not a git repository; run this command from inside a repo")
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

	suggestion := suggestRule(cfg, cwd, remoteURL, currentEmail)
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
	fmt.Println(ui.SuccessStyle.Render(fmt.Sprintf("Suggested: use identity '%s'", suggestion.Identity)))
	fmt.Println()
	fmt.Println("Run this to add the rule:")
	fmt.Printf("  gitch rule add --remote %q --use %s\n", suggestion.Pattern, suggestion.Identity)

	return nil
}
