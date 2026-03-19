package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/profile"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var (
	checkJSON bool
	checkFix  bool
)

// checkOutput represents the JSON output structure for gitch check --json
type checkOutput struct {
	OK              bool   `json:"ok"`
	Fixed           bool   `json:"fixed,omitempty"`
	CurrentName     string `json:"current_name,omitempty"`
	CurrentEmail    string `json:"current_email,omitempty"`
	ExpectedName    string `json:"expected_name,omitempty"`
	ExpectedEmail   string `json:"expected_email,omitempty"`
	RuleType        string `json:"rule_type,omitempty"`
	RulePattern     string `json:"rule_pattern,omitempty"`
	ExpectedProfile string `json:"expected_profile,omitempty"`
	Reason          string `json:"reason"`
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Verify the current identity matches the expected rule",
	Long: `Check whether the current git identity matches the identity expected by
the best-matching rule for this directory or remote.

Exit code 0 means the identity is correct or no rule applies.
Exit code 1 means the active identity does not match the expected rule.

Use --fix to automatically apply the correct identity when a mismatch is detected.
With --fix, exit code 0 means the identity was already correct or was successfully fixed.

Use this in scripts, CI pipelines, or as a pre-commit guard:

  gitch check && git commit -m "safe to commit"
  gitch check --fix && git commit -m "auto-fixed and safe"

Examples:
  gitch check
  gitch check --fix
  gitch check --json`,
	Args:          cobra.NoArgs,
	RunE:          runCheck,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output in JSON format")
	checkCmd.Flags().BoolVar(&checkFix, "fix", false, "Automatically apply the correct identity when a mismatch is detected")
}

func runCheck(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	matchedRule := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	// No rule applies — nothing to check against
	if matchedRule == nil {
		out := checkOutput{
			OK:     true,
			Reason: "no rule matches this directory",
		}
		return printCheckResult(out)
	}

	expectedIdentity, err := cfg.GetIdentity(matchedRule.Identity)
	if err != nil {
		out := checkOutput{
			OK:          false,
			RuleType:    string(matchedRule.Type),
			RulePattern: matchedRule.Pattern,
			Reason:      "rule references unknown identity '" + matchedRule.Identity + "'",
		}
		return printCheckResult(out)
	}

	matches, matchErr := profile.Matches(expectedIdentity)

	currentName, currentEmail, _ := gitpkg.GetCurrentIdentity()

	out := checkOutput{
		CurrentName:     currentName,
		CurrentEmail:    currentEmail,
		ExpectedName:    expectedIdentity.GitAuthorName(),
		ExpectedEmail:   expectedIdentity.Email,
		RuleType:        string(matchedRule.Type),
		RulePattern:     matchedRule.Pattern,
		ExpectedProfile: expectedIdentity.Name,
	}

	if matchErr == nil && matches {
		out.OK = true
		out.Reason = "identity matches rule"
		return printCheckResult(out)
	}

	// Mismatch detected — fix if requested
	if checkFix {
		scope := defaultApplyScope()
		if err := applyConfiguredIdentity(expectedIdentity, scope); err != nil {
			out.OK = false
			out.Reason = "fix failed: " + err.Error()
			return printCheckResult(out)
		}
		out.OK = true
		out.Fixed = true
		out.Reason = "identity mismatch fixed"
		return printCheckResult(out)
	}

	out.OK = false
	out.Reason = "identity does not match rule"
	return printCheckResult(out)
}

func printCheckResult(out checkOutput) error {
	if checkJSON {
		if err := printJSON(out); err != nil {
			return err
		}
		if !out.OK {
			return errors.New(out.Reason)
		}
		return nil
	}

	if out.OK {
		if out.Fixed {
			fmt.Printf("%s Applied identity '%s' (%s) to match %s rule '%s'.\n",
				ui.SuccessStyle.Render("FIXED"),
				out.ExpectedProfile,
				out.ExpectedEmail,
				out.RuleType,
				out.RulePattern)
		} else if out.ExpectedProfile != "" {
			fmt.Printf("%s Identity '%s' (%s) matches %s rule '%s'.\n",
				ui.SuccessStyle.Render("OK"),
				out.ExpectedProfile,
				out.ExpectedEmail,
				out.RuleType,
				out.RulePattern)
		} else {
			fmt.Println(ui.DimStyle.Render("No rule matches this directory — nothing to check."))
		}
		return nil
	}

	// Failure case
	if out.ExpectedEmail == "" {
		// Rule references unknown identity
		fmt.Printf("%s %s\n", ui.ErrorStyle.Render("FAIL"), out.Reason)
		return errors.New(out.Reason)
	}

	fmt.Printf("%s Identity mismatch for %s rule '%s'.\n",
		ui.ErrorStyle.Render("FAIL"),
		out.RuleType,
		out.RulePattern)

	if out.CurrentEmail == "" {
		fmt.Println("  Current:  no identity configured")
	} else {
		fmt.Println("  Current:  " + out.CurrentName + " (" + out.CurrentEmail + ")")
	}
	fmt.Println("  Expected: " + out.ExpectedName + " (" + out.ExpectedEmail + ")")
	fmt.Println()
	fmt.Println("Fix: gitch use " + out.ExpectedProfile)

	return errors.New("identity does not match rule")
}
