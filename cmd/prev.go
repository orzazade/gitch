package cmd

import (
	"errors"
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	"github.com/spf13/cobra"
)

var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Switch back to the previously used identity",
	Long: `Switch back to the identity that was active before the last switch.

Works like 'cd -' for git identities. Each time you run 'gitch use' or
'gitch autoswitch', the previous identity is saved. Running 'gitch prev'
restores it.

Running 'gitch prev' twice toggles between the two most recent identities.

Examples:
  gitch use work       # switch to work
  gitch use personal   # switch to personal
  gitch prev           # back to work
  gitch prev           # back to personal`,
	Args:          cobra.NoArgs,
	RunE:          runPrev,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(prevCmd)
}

func runPrev(cmd *cobra.Command, args []string) error {
	prevName, err := config.LoadPreviousIdentity()
	if err != nil {
		return fmt.Errorf("failed to load previous identity: %w", err)
	}
	if prevName == "" {
		return errors.New("no previous identity — switch identities first with 'gitch use'")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	identity, err := cfg.GetIdentity(prevName)
	if err != nil {
		return fmt.Errorf("previous identity '%s' no longer exists — it may have been deleted or renamed", prevName)
	}

	// Save current identity as prev before switching (enables toggle behavior).
	savePrevIdentity(cfg)

	scope := defaultApplyScope()
	if err := applyConfiguredIdentity(identity, scope); err != nil {
		return fmt.Errorf("failed to switch identity: %w", err)
	}

	printSwitchSuccess(identity)
	printScopeInfo(scope)
	return nil
}
