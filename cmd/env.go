package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var envJSON bool

var errEnvConflict = errors.New("environment variables conflict with expected identity")

type envVarInfo struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Expected string `json:"expected,omitempty"`
	Conflict bool   `json:"conflict"`
}

type envOutput struct {
	Variables       []envVarInfo `json:"variables"`
	ExpectedName    string       `json:"expected_name,omitempty"`
	ExpectedEmail   string       `json:"expected_email,omitempty"`
	ExpectedProfile string       `json:"expected_profile,omitempty"`
	HasConflict     bool         `json:"has_conflict"`
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show git identity environment variables and flag conflicts",
	Long: `Show environment variables that affect git identity and warn if they
conflict with the expected gitch identity.

Git uses GIT_AUTHOR_EMAIL, GIT_COMMITTER_EMAIL, GIT_AUTHOR_NAME, and
GIT_COMMITTER_NAME to override user.email and user.name from git config.
These overrides are invisible to gitch and can cause wrong-identity commits
even when gitch is correctly configured.

Exit code 0 means no conflicts. Exit code 1 means env vars override the
expected identity.

Examples:
  gitch env
  gitch env --json`,
	Args:          cobra.NoArgs,
	RunE:          runEnv,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().BoolVar(&envJSON, "json", false, "Output in JSON format")
}

// identityEnvVars are the environment variables that override git identity.
var identityEnvVars = []string{
	"GIT_AUTHOR_NAME",
	"GIT_AUTHOR_EMAIL",
	"GIT_COMMITTER_NAME",
	"GIT_COMMITTER_EMAIL",
}

func runEnv(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	expectedName, expectedEmail, profileName := resolveExpectedIdentityForEnv(cfg)

	out := envOutput{
		ExpectedName:    expectedName,
		ExpectedEmail:   expectedEmail,
		ExpectedProfile: profileName,
	}

	for _, varName := range identityEnvVars {
		value, set := os.LookupEnv(varName)
		if !set {
			continue
		}

		ev := envVarInfo{
			Name:  varName,
			Value: value,
		}

		switch varName {
		case "GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME":
			if expectedName != "" && !strings.EqualFold(value, expectedName) {
				ev.Conflict = true
				ev.Expected = expectedName
				out.HasConflict = true
			}
		case "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL":
			if expectedEmail != "" && !strings.EqualFold(value, expectedEmail) {
				ev.Conflict = true
				ev.Expected = expectedEmail
				out.HasConflict = true
			}
		}

		out.Variables = append(out.Variables, ev)
	}

	if envJSON {
		if err := printJSON(out); err != nil {
			return err
		}
		if out.HasConflict {
			return errEnvConflict
		}
		return nil
	}

	return printEnvHuman(out)
}

func resolveExpectedIdentityForEnv(cfg *config.Config) (name, email, profileName string) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	remoteURL, _ := rules.GetGitRemoteURL()

	if matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matched != nil {
		if identity, err := cfg.GetIdentity(matched.Identity); err == nil {
			return identity.GitAuthorName(), identity.Email, identity.Name
		}
	}

	state, err := resolveCurrentProfileState(cfg)
	if err != nil {
		return "", "", ""
	}

	if state.ExactMatch != nil {
		return state.ExactMatch.GitAuthorName(), state.ExactMatch.Email, state.ExactMatch.Name
	}
	if state.EmailMatch != nil {
		return state.EmailMatch.GitAuthorName(), state.EmailMatch.Email, state.EmailMatch.Name
	}

	return state.CurrentName, state.CurrentEmail, ""
}

func printEnvHuman(out envOutput) error {
	fmt.Println(ui.DimStyle.Render("gitch env — identity environment check"))
	fmt.Println()

	if len(out.Variables) == 0 {
		fmt.Println("  No git identity environment variables set.")
		if out.ExpectedProfile != "" {
			fmt.Printf("\n  Expected identity: %s (%s)\n", out.ExpectedProfile, out.ExpectedEmail)
		}
		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("No environment overrides — git config is in control."))
		return nil
	}

	if out.ExpectedProfile != "" {
		fmt.Printf("  Expected: %s (%s)\n", out.ExpectedProfile, out.ExpectedEmail)
		fmt.Println()
	}

	for _, ev := range out.Variables {
		if ev.Conflict {
			fmt.Printf("  %s  %s=%s\n", ui.WarningStyle.Render("conflict"), ev.Name, ev.Value)
			fmt.Printf("             %s\n", ui.DimStyle.Render("expected: "+ev.Expected))
		} else {
			fmt.Printf("  %s       %s=%s\n", ui.SuccessStyle.Render("ok"), ev.Name, ev.Value)
		}
	}

	fmt.Println()
	if out.HasConflict {
		fmt.Println(ui.WarningStyle.Render("Environment variables override git config and may cause wrong-identity commits."))
		fmt.Println()
		fmt.Println("Fix: unset conflicting variables, or use 'gitch exec' to run commands with the correct identity.")
		return errEnvConflict
	}

	fmt.Println(ui.SuccessStyle.Render("Environment variables match the expected identity."))
	return nil
}
