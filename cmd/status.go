package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var statusVerbose bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current active git identity",
	Long: `Show the currently active git identity.

Displays the name and email from git config and indicates if it's
managed by gitch.

Use -v to show which rule matches the current directory/remote.

Examples:
  gitch status
  gitch status -v`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show matched rule details")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Get current git identity
	name, email, err := git.GetCurrentIdentity()
	if err != nil {
		return fmt.Errorf("failed to get current identity: %w", err)
	}

	// Check if git has any identity configured
	if name == "" && email == "" {
		fmt.Println("No active identity. Use 'gitch use <name>' to set one.")
		return nil
	}

	// Load config to check if this identity is managed by gitch
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Try to find matching identity by email
	var managed bool
	var managedName string
	for _, identity := range cfg.ListIdentities() {
		if strings.EqualFold(identity.Email, email) {
			managed = true
			managedName = identity.Name
			break
		}
	}

	// Format output
	if managed {
		msg := fmt.Sprintf("Active: %s (%s)", managedName, email)
		fmt.Println(ui.SuccessStyle.Render(msg))
	} else {
		if name != "" {
			fmt.Printf("Active: %s (%s) ", name, email)
		} else {
			fmt.Printf("Active: (%s) ", email)
		}
		fmt.Println(ui.WarningStyle.Render("[not managed by gitch]"))
	}

	// Show verbose rule matching information
	if statusVerbose {
		showVerboseRuleInfo(cfg, email)
	}

	return nil
}

// showVerboseRuleInfo displays which rule matches the current directory/remote
func showVerboseRuleInfo(cfg *config.Config, currentEmail string) {
	// Get current directory and remote
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	// Find matching rule
	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	fmt.Println()
	if matchedRule != nil {
		fmt.Println(ui.DimStyle.Render("Matched rule:"))
		fmt.Printf("  Type:     %s\n", matchedRule.Type)
		fmt.Printf("  Pattern:  %s\n", matchedRule.Pattern)
		fmt.Printf("  Identity: %s\n", matchedRule.Identity)

		// Check if current identity matches expected
		expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
		if err == nil && expectedIdentity != nil {
			if !strings.EqualFold(currentEmail, expectedIdentity.Email) {
				fmt.Println()
				fmt.Println(ui.WarningStyle.Render("Warning: Current identity does not match rule!"))
				fmt.Printf("  Expected: %s (%s)\n", expectedIdentity.Name, expectedIdentity.Email)
			}
		}
	} else {
		fmt.Println(ui.DimStyle.Render("No rule matched for current directory"))
	}
}
