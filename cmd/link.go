package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	gitpkg "github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/hooks"
	"github.com/orzazade/gitch/internal/rules"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <identity>",
	Short: "Set up identity management for this repository in one step",
	Long: `Link an identity to the current repository.

This is the fastest way to set up a new repo with gitch. In one command it:
  1. Applies the identity to local git config
  2. Creates a directory rule for this repo (if no rule already matches)
  3. Installs the pre-commit hook (if not already installed)

Equivalent to running:
  gitch use --local <identity>
  gitch rule add <repo-path>/** --use <identity>
  gitch hook install

Examples:
  cd ~/work/project && gitch link work
  cd ~/personal/blog && gitch link personal`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: identityCompletionFunc,
	RunE:              runLink,
}

func init() {
	rootCmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, args []string) error {
	if !gitpkg.IsGitRepository() {
		return errNotGitRepo
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	name := args[0]
	identity, err := cfg.GetIdentity(name)
	if err != nil {
		msg := "identity '" + name + "' not found"
		if suggestion := closestIdentityName(name, cfg.ListIdentities()); suggestion != "" {
			msg += "\n\nDid you mean '" + suggestion + "'?  Run: gitch link " + suggestion
		}
		return errors.New(msg)
	}

	repoRoot, err := gitpkg.Toplevel()
	if err != nil {
		return fmt.Errorf("failed to determine repository root: %w", err)
	}

	// 1. Apply identity to local git config.
	savePrevIdentity(cfg)
	if err := applyConfiguredIdentity(identity, gitpkg.ScopeLocal); err != nil {
		return fmt.Errorf("failed to apply identity: %w", err)
	}
	printSwitchSuccess(identity)

	// 2. Create a directory rule if no rule already matches this repo.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	remoteURL, _ := rules.GetGitRemoteURL()
	matched := rules.FindBestMatch(cfg.Rules, cwd, remoteURL)

	if matched != nil && strings.EqualFold(matched.Identity, identity.Name) {
		fmt.Println(ui.DimStyle.Render("Rule: already covered by " + matched.Pattern))
	} else {
		home, _ := os.UserHomeDir()
		pattern := shortenHome(repoRoot, home) + "/**"

		rule := rules.Rule{
			Type:     rules.DirectoryRule,
			Pattern:  pattern,
			Identity: identity.Name,
		}

		if matched != nil {
			fmt.Println(ui.WarningStyle.Render("Note: existing rule " + matched.Pattern + " -> " + matched.Identity + " also matches this directory"))
		}

		if err := cfg.AddRule(rule); err != nil {
			return fmt.Errorf("failed to add rule: %w", err)
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Rule added: " + pattern + " -> " + identity.Name))
	}

	// 3. Install pre-commit hook if not already installed.
	hookInstalled, err := hooks.IsInstalledLocal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check hook status: %v\n", err)
	} else if hookInstalled {
		fmt.Println(ui.DimStyle.Render("Hook: pre-commit already installed"))
	} else {
		if err := hooks.InstallLocal(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to install hook: %v\n", err)
		} else {
			fmt.Println(ui.SuccessStyle.Render("Hook: pre-commit installed"))
		}
	}

	return nil
}

