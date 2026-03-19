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

var remoteJSON bool

var errRemoteConflict = errors.New("remotes expect different identities")

type remoteInfo struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Identity string `json:"identity,omitempty"`
	Email    string `json:"email,omitempty"`
	RuleType string `json:"rule_type,omitempty"`
	Pattern  string `json:"rule_pattern,omitempty"`
	Matched  bool   `json:"matched"`
	Current  bool   `json:"current"`
}

type remoteOutput struct {
	Remotes         []remoteInfo `json:"remotes"`
	CurrentIdentity string       `json:"current_identity,omitempty"`
	CurrentEmail    string       `json:"current_email,omitempty"`
	HasConflict     bool         `json:"has_conflict"`
}

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Show identity expectations per remote",
	Long: `Show all git remotes in the current repository and which identity
each one expects based on your rules.

This is especially useful for fork workflows where different remotes
(e.g., origin vs upstream) may belong to different organizations and
expect different identities.

Exit code 0 means all remotes expect the same identity (or no rules match).
Exit code 1 means different remotes expect different identities.

Examples:
  gitch remote
  gitch remote --json`,
	Args:          cobra.NoArgs,
	RunE:          runRemote,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.Flags().BoolVar(&remoteJSON, "json", false, "Output in JSON format")
}

func runRemote(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	remotes, err := gitpkg.Remotes()
	if err != nil {
		return fmt.Errorf("failed to list remotes: %w", err)
	}

	if len(remotes) == 0 {
		if remoteJSON {
			return printJSON(remoteOutput{})
		}
		fmt.Println("No remotes configured in this repository.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	_, currentEmail, _ := gitpkg.GetCurrentIdentity()

	out := remoteOutput{
		CurrentEmail: currentEmail,
	}

	// Resolve current identity name from config.
	for i := range cfg.Identities {
		if strings.EqualFold(cfg.Identities[i].Email, currentEmail) {
			out.CurrentIdentity = cfg.Identities[i].Name
			break
		}
	}

	identities := make(map[string]bool)

	for _, r := range remotes {
		ri := remoteInfo{
			Name: r.Name,
			URL:  r.URL,
		}

		// Try matching with this remote's URL.
		matched := rules.FindBestMatch(cfg.Rules, cwd, r.URL)
		if matched != nil {
			ri.Matched = true
			ri.RuleType = string(matched.Type)
			ri.Pattern = matched.Pattern
			ri.Identity = matched.Identity

			if identity, err := cfg.GetIdentity(matched.Identity); err == nil {
				ri.Email = identity.Email
				ri.Current = strings.EqualFold(identity.Email, currentEmail)
			}

			identities[matched.Identity] = true
		}

		out.Remotes = append(out.Remotes, ri)
	}

	// Conflict if multiple different identities are expected.
	if len(identities) > 1 {
		out.HasConflict = true
	}

	if remoteJSON {
		if err := printJSON(out); err != nil {
			return err
		}
		if out.HasConflict {
			return errRemoteConflict
		}
		return nil
	}

	return printRemoteHuman(out)
}

func printRemoteHuman(out remoteOutput) error {
	fmt.Println(ui.DimStyle.Render("gitch remote — identity expectations per remote"))
	fmt.Println()

	if out.CurrentIdentity != "" {
		fmt.Printf("  Active: %s (%s)\n", out.CurrentIdentity, out.CurrentEmail)
	} else if out.CurrentEmail != "" {
		fmt.Printf("  Active: %s\n", out.CurrentEmail)
	}
	fmt.Println()

	for _, r := range out.Remotes {
		if !r.Matched {
			fmt.Printf("  %-12s %s\n", r.Name, r.URL)
			fmt.Printf("  %s    %s\n", " ", ui.DimStyle.Render("no matching rule"))
			continue
		}

		label := r.Identity
		if r.Email != "" {
			label += " (" + r.Email + ")"
		}

		if r.Current {
			fmt.Printf("  %-12s %s\n", r.Name, r.URL)
			fmt.Printf("  %s    %s %s\n", " ", ui.SuccessStyle.Render("ok"), label)
		} else {
			fmt.Printf("  %-12s %s\n", r.Name, r.URL)
			fmt.Printf("  %s    %s %s\n", " ", ui.WarningStyle.Render("expects"), label)
		}
	}

	fmt.Println()
	if out.HasConflict {
		fmt.Println(ui.WarningStyle.Render("Different remotes expect different identities."))
		fmt.Println()
		fmt.Println("Note: git commits carry one identity. Pushes to remotes expecting")
		fmt.Println("a different identity will use your current identity, not the remote's.")
		fmt.Println("Consider switching identity before pushing: gitch use <identity> && git push <remote>")
		return errRemoteConflict
	}

	return nil
}
