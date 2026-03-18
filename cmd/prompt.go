package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Print identity name for shell prompt integration",
	Long: `Output a compact identity indicator suitable for embedding in your shell prompt.

The output tells you at a glance which git identity is active:
  - Managed identity fully applied:  identity name (e.g. "work")
  - Email matches but not fully applied:  name with ~ suffix (e.g. "work~")
  - Git identity not managed by gitch:  "?"
  - No git identity configured:  empty (no output)
  - Not in a git repo:  empty (no output)

All errors produce empty output — prompt commands must never break your terminal.

Examples:
  # bash — add to PS1
  PS1='$(gitch prompt) \w $ '

  # zsh — add to PROMPT
  PROMPT='$(gitch prompt) %~ %# '

  # fish — in fish_prompt function
  echo -n (gitch prompt)" "

  # starship — custom command
  [custom.gitch]
  command = "gitch prompt"
  when = "command -v gitch"`,
	Args:          cobra.NoArgs,
	Run:           runPrompt,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) {
	label := resolvePromptLabel()
	if label != "" {
		fmt.Print(label)
	}
}

// resolvePromptLabel determines the identity label for the shell prompt.
// Returns empty string when there's nothing useful to show.
func resolvePromptLabel() string {
	if !gitpkg.IsGitRepository() {
		return ""
	}

	_, email, err := gitpkg.GetCurrentIdentity()
	if err != nil || email == "" {
		return ""
	}

	cfg, err := config.Load()
	if err != nil || len(cfg.Identities) == 0 {
		return "?"
	}

	// Check for rule-based expected identity first.
	if label := resolvePromptFromRule(cfg, email); label != "" {
		return label
	}

	// Fall back to matching against all managed identities.
	return resolvePromptFromIdentities(cfg, email)
}

// resolvePromptFromRule checks if a rule matches and whether the current
// identity satisfies it. Returns empty string if no rule applies.
func resolvePromptFromRule(cfg *config.Config, currentEmail string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	remoteURL, _ := rules.GetGitRemoteURL()
	matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)
	if matched == nil {
		return ""
	}

	expected, err := cfg.GetIdentity(matched.Identity)
	if err != nil {
		return ""
	}

	if strings.EqualFold(currentEmail, expected.Email) {
		return expected.Name
	}
	// Rule expects a different identity — show expected name with ! warning.
	return expected.Name + "!"
}

// resolvePromptFromIdentities matches the current email against all managed
// identities. Returns the identity name, or "?" if unmanaged.
func resolvePromptFromIdentities(cfg *config.Config, currentEmail string) string {
	for i := range cfg.Identities {
		if strings.EqualFold(cfg.Identities[i].Email, currentEmail) {
			return cfg.Identities[i].Name
		}
	}
	return "?"
}
