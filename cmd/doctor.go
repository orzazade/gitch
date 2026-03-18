package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/gpg"
	"github.com/orzazade/gitch/internal/hooks"
	"github.com/orzazade/gitch/internal/rules"
	sshpkg "github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check gitch setup for common problems",
	Long: `Run health checks on your gitch setup and report issues with fix instructions.

Exits with code 1 if any issues are found, so it can be used in scripts:
  gitch doctor && echo "ready to commit"

Examples:
  gitch doctor`,
	RunE:          runDoctor,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type doctorCheck struct {
	ok    bool
	label string
	fix   string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var checks []doctorCheck

	cfg, err := config.Load()
	if err != nil {
		checks = append(checks, doctorCheck{false, "cannot load gitch config", "check ~/.config/gitch/config.yaml for syntax errors"})
		if printDoctorResults(checks) > 0 {
			return errors.New("doctor found issues")
		}
		return nil
	}

	n := len(cfg.Identities)
	if n == 0 {
		checks = append(checks, doctorCheck{false, "no identities configured", "run: gitch add"})
		if printDoctorResults(checks) > 0 {
			return errors.New("doctor found issues")
		}
		return nil
	}
	checks = append(checks, doctorCheck{true, strconv.Itoa(n) + " " + nounPlural(n, "identity", "identities") + " configured", ""})

	state, err := resolveCurrentProfileState(cfg)
	if err != nil || (state.CurrentName == "" && state.CurrentEmail == "") {
		checks = append(checks, doctorCheck{false, "no active identity in git config", "run: gitch use <name>"})
	} else {
		identity := state.ExactMatch
		if identity == nil {
			identity = state.EmailMatch
		}

		if identity == nil {
			checks = append(checks, doctorCheck{false, fmt.Sprintf("active email %q is not managed by gitch", state.CurrentEmail), "run: gitch use <name>"})
		} else {
			checks = append(checks, doctorCheck{true, "active identity: " + identity.Name + " (" + identity.Email + ")", ""})

			if identity.SSHKeyPath != "" {
				keyPath, expandErr := sshpkg.ExpandPath(identity.SSHKeyPath)
				if expandErr != nil {
					checks = append(checks, doctorCheck{false, "invalid SSH key path: " + identity.SSHKeyPath, "run: gitch edit " + identity.Name + " --ssh-key <path>"})
				} else if _, statErr := os.Stat(keyPath); statErr != nil {
					checks = append(checks, doctorCheck{false, "SSH key file missing: " + identity.SSHKeyPath, "run: gitch add to reconfigure '" + identity.Name + "'"})
				} else {
					checks = append(checks, doctorCheck{true, "SSH key exists: " + identity.SSHKeyPath, ""})
					if !sshpkg.IsAgentRunning() {
						checks = append(checks, doctorCheck{false, "ssh-agent not running", "run: eval $(ssh-agent) && ssh-add " + keyPath})
					} else if !sshpkg.IsKeyLoadedInAgent(keyPath) {
						checks = append(checks, doctorCheck{false, "SSH key not loaded in agent", "run: ssh-add " + keyPath})
					} else {
						checks = append(checks, doctorCheck{true, "SSH key loaded in agent", ""})
					}
				}
			}

			if identity.GPGKeyID != "" {
				if !gpg.IsGPGAvailable() {
					checks = append(checks, doctorCheck{false, "gpg not installed (required for commit signing)", "install GPG: sudo apt install gnupg"})
				} else if gpgErr := gpg.ValidateKeyID(identity.GPGKeyID); gpgErr != nil {
					checks = append(checks, doctorCheck{false, "GPG key not found in keyring: " + identity.GPGKeyID, "import the key or run: gitch add to reconfigure"})
				} else {
					checks = append(checks, doctorCheck{true, "GPG key valid: " + identity.GPGKeyID, ""})
				}
			}
		}
	}

	globalOK, _ := hooks.IsInstalled()
	localOK, _ := hooks.IsInstalledLocal()
	switch {
	case globalOK:
		checks = append(checks, doctorCheck{true, "pre-commit hook installed (global)", ""})
	case localOK:
		checks = append(checks, doctorCheck{true, "pre-commit hook installed (local)", ""})
	default:
		checks = append(checks, doctorCheck{false, "no pre-commit hook installed — commits are unguarded", "run: gitch hook install"})
	}

	if len(cfg.Rules) > 0 {
		cwd, _ := os.Getwd()
		remoteURL, _ := rules.GetGitRemoteURL()
		if matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL); matched != nil {
			checks = append(checks, doctorCheck{true, "auto-switch rule matched: " + matched.Pattern + " → " + matched.Identity, ""})
		} else {
			checks = append(checks, doctorCheck{false, "no auto-switch rule matches current directory", "run: gitch rule add"})
		}
	}

	if printDoctorResults(checks) > 0 {
		return errors.New("doctor found issues")
	}
	return nil
}

func printDoctorResults(checks []doctorCheck) int {
	fmt.Println(ui.DimStyle.Render("gitch doctor — health check"))
	fmt.Println()
	warnings := 0
	for _, c := range checks {
		if c.ok {
			fmt.Printf("  %s  %s\n", ui.SuccessStyle.Render("ok  "), c.label)
		} else {
			warnings++
			fmt.Printf("  %s  %s\n", ui.WarningStyle.Render("warn"), c.label)
			if c.fix != "" {
				fmt.Printf("        %s\n", ui.DimStyle.Render(c.fix))
			}
		}
	}
	fmt.Println()
	if warnings == 0 {
		fmt.Println(ui.SuccessStyle.Render("Everything looks good."))
	} else {
		fmt.Println(ui.WarningStyle.Render(strconv.Itoa(warnings) + " " + nounPlural(warnings, "issue", "issues") + " found."))
	}
	return warnings
}

func nounPlural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
