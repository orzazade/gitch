package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var statusVerbose bool
var statusJSON bool
var statusAutoSwitch bool

// statusOutput represents the JSON output structure for gitch status --json
type statusOutput struct {
	Name         string `json:"name"`
	GitName      string `json:"git_name,omitempty"`
	Email        string `json:"email"`
	SSHKeyPath   string `json:"ssh_key_path,omitempty"`
	GPGKeyID     string `json:"gpg_key_id,omitempty"`
	Managed      bool   `json:"managed"`
	PartialMatch bool   `json:"partial_match,omitempty"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current active git identity",
	Long: `Show the currently active git identity.

Displays the name and email from git config and indicates if it's
managed by gitch.

Use -v to show which rule matches the current directory/remote.
Use --auto-switch to apply a matching rule before printing status.

Examples:
  gitch status
  gitch status -v`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show matched rule details")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output in JSON format")
	statusCmd.Flags().BoolVar(&statusAutoSwitch, "auto-switch", false, "Apply a matching rule before showing status")
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load config first for status resolution
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply auto-switch only when explicitly requested.
	if statusAutoSwitch {
		result, err := tryAutoSwitch(cfg)
		if err != nil {
			return fmt.Errorf("failed to auto-switch: %w", err)
		}
		if result.Switched && !statusJSON {
			fmt.Println(ui.SuccessStyle.Render("Switched to '" + result.ToIdentity + "' identity"))
			fmt.Println()
		}
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve current identity: %w", err)
	}
	name := state.CurrentName
	email := state.CurrentEmail

	// Check if git has any identity configured
	if name == "" && email == "" {
		if statusJSON {
			// Output empty JSON for no identity case
			output := statusOutput{}
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(jsonBytes))
			return nil
		}
		fmt.Println("No active identity. Use 'gitch use <name>' to set one.")
		return nil
	}

	managedIdentity := state.ExactMatch
	emailMatchedIdentity := state.EmailMatch
	managed := managedIdentity != nil
	partialMatch := !managed && emailMatchedIdentity != nil

	// JSON output format
	if statusJSON {
		output := statusOutput{
			Email:        email,
			Managed:      managed,
			PartialMatch: partialMatch,
		}
		switch {
		case managedIdentity != nil:
			output.Name = managedIdentity.Name
			output.GitName = managedIdentity.GitAuthorName()
			output.SSHKeyPath = managedIdentity.SSHKeyPath
			output.GPGKeyID = managedIdentity.GPGKeyID
		case emailMatchedIdentity != nil:
			output.Name = emailMatchedIdentity.Name
			output.GitName = name
			output.SSHKeyPath = emailMatchedIdentity.SSHKeyPath
			output.GPGKeyID = emailMatchedIdentity.GPGKeyID
		case name != "":
			output.Name = name
			output.GitName = name
		}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Format output
	if managed {
		fmt.Println(ui.SuccessStyle.Render("Active: " + managedIdentity.Name + " (" + email + ")"))
		fmt.Printf("Git Author: %s\n", managedIdentity.GitAuthorName())
		if managedIdentity.GPGKeyID != "" {
			fmt.Printf("GPG Key: %s\n", managedIdentity.GPGKeyID)
		}
	} else if partialMatch {
		fmt.Printf("Active: %s (%s) ", name, email)
		fmt.Println(ui.WarningStyle.Render("[profile '" + emailMatchedIdentity.Name + "' not fully applied]"))
		fmt.Printf("Expected Profile: %s\n", emailMatchedIdentity.Name)
		fmt.Printf("Expected Git Author: %s\n", emailMatchedIdentity.GitAuthorName())
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
		if err == nil {
			match, matchErr := profile.Matches(expectedIdentity)
			if (matchErr == nil && !match) || (matchErr != nil && !strings.EqualFold(currentEmail, expectedIdentity.Email)) {
				fmt.Println()
				fmt.Println(ui.WarningStyle.Render("Warning: Current identity does not match rule!"))
				fmt.Printf("  Expected: %s (%s)\n", expectedIdentity.GitAuthorName(), expectedIdentity.Email)
			}
		}
	} else {
		fmt.Println(ui.DimStyle.Render("No rule matched for current directory"))
	}
}
