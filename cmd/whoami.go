package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var whoamiJSON bool

type whoamiOutput struct {
	Name        string `json:"name"`
	GitName     string `json:"git_name"`
	Email       string `json:"email"`
	Managed     bool   `json:"managed"`
	ExpectedBy  string `json:"expected_by,omitempty"`
	RulePattern string `json:"rule_pattern,omitempty"`
	Mismatch    bool   `json:"mismatch,omitempty"`
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Print the current git identity in author format",
	Long: `Show the active git identity as a single line in git author format:
  Name <email>

This is the quickest way to check who you are in the current repository.
Use --json for machine-readable output.

Exit code 0: identity is configured.
Exit code 1: no identity is configured or not in a git repository.

Examples:
  gitch whoami
  gitch whoami --json
  gitch whoami && git commit -m "safe"`,
	Args:          cobra.NoArgs,
	RunE:          runWhoami,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
	whoamiCmd.Flags().BoolVar(&whoamiJSON, "json", false, "Output in JSON format")
}

func runWhoami(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	currentName, currentEmail, err := gitpkg.GetCurrentIdentity()
	if err != nil {
		return fmt.Errorf("failed to read git identity: %w", err)
	}

	if currentName == "" && currentEmail == "" {
		if whoamiJSON {
			return printJSON(whoamiOutput{})
		}
		return errors.New("no identity configured — run 'gitch use <name>' to set one")
	}

	// Check if this email belongs to a managed identity.
	cfg, _ := config.Load()
	managed := false
	identityName := ""
	if cfg != nil {
		for _, id := range cfg.Identities {
			if strings.EqualFold(id.Email, currentEmail) {
				managed = true
				identityName = id.Name
				break
			}
		}
	}

	out := whoamiOutput{
		Name:    identityName,
		GitName: currentName,
		Email:   currentEmail,
		Managed: managed,
	}

	// Check for rule mismatch.
	if cfg != nil && len(cfg.Rules) > 0 {
		cwd, cwdErr := os.Getwd()
		if cwdErr == nil {
			remoteURL, _ := rules.GetGitRemoteURL()
			if matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matched != nil {
				if expected, getErr := cfg.GetIdentity(matched.Identity); getErr == nil {
					if !strings.EqualFold(currentEmail, expected.Email) {
						out.Mismatch = true
						out.ExpectedBy = expected.Name
						out.RulePattern = matched.Pattern
					}
				}
			}
		}
	}

	if whoamiJSON {
		return printJSON(out)
	}

	// Print git author format line.
	if currentName != "" {
		fmt.Printf("%s <%s>\n", currentName, currentEmail)
	} else {
		fmt.Printf("<%s>\n", currentEmail)
	}

	if out.Mismatch {
		fmt.Fprintln(os.Stderr, ui.WarningStyle.Render(
			"expected '"+out.ExpectedBy+"' (rule: "+out.RulePattern+")"))
	}

	return nil
}
