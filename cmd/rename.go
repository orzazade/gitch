package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a git identity",
	Long: `Rename a git identity, updating all references.

Any rules pointing to the old name are updated automatically.
If the renamed identity is the default, the default is updated too.

Examples:
  gitch rename personal github
  gitch rename work company`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: identityCompletionFunc,
	RunE:              runRename,
}

func init() {
	rootCmd.AddCommand(renameCmd)
}

func runRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Verify old identity exists before renaming (for a better error message)
	oldIdentity, err := cfg.GetIdentity(oldName)
	if err != nil {
		msg := "identity '" + oldName + "' not found"
		if suggestion := closestIdentityName(oldName, cfg.ListIdentities()); suggestion != "" {
			msg += "\n\nDid you mean '" + suggestion + "'?"
		}
		return errors.New(msg)
	}
	actualOldName := oldIdentity.Name

	if err := cfg.RenameIdentity(oldName, newName); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update prev-identity file if it references the old name.
	if prevName, err := config.LoadPreviousIdentity(); err == nil && strings.EqualFold(prevName, actualOldName) {
		_ = config.SavePreviousIdentity(newName)
	}

	fmt.Println(ui.SuccessStyle.Render("Renamed '" + actualOldName + "' to '" + newName + "'"))

	// Show how many rules were updated
	updatedRules := 0
	for _, r := range cfg.Rules {
		if r.Identity == newName {
			updatedRules++
		}
	}
	if updatedRules > 0 {
		fmt.Printf("Updated %d rule(s) to reference '%s'\n", updatedRules, newName)
	}

	return nil
}
