package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/git"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

// listOutputItem represents a single identity in JSON output
type listOutputItem struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	SSHKeyPath string `json:"ssh_key_path,omitempty"`
	GPGKeyID   string `json:"gpg_key_id,omitempty"`
	IsActive   bool   `json:"is_active"`
	IsDefault  bool   `json:"is_default"`
}

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured identities",
	Long: `List all configured git identities.

The currently active identity is highlighted with a checkmark and green border.
The default identity is marked with "(default)".

Examples:
  gitch list
  gitch ls`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all identities
	identities := cfg.ListIdentities()
	if len(identities) == 0 {
		fmt.Println("No identities configured. Use 'gitch add' to create one.")
		return nil
	}

	// Get current git identity to determine which is active
	_, activeEmail, err := git.GetCurrentIdentity()
	if err != nil {
		// Non-fatal: just means no identity will be marked as active
		activeEmail = ""
	}

	// JSON output for machine consumption
	if listJSON {
		items := make([]listOutputItem, len(identities))
		for i, id := range identities {
			items[i] = listOutputItem{
				Name:       id.Name,
				Email:      id.Email,
				SSHKeyPath: id.SSHKeyPath,
				GPGKeyID:   id.GPGKeyID,
				IsActive:   strings.EqualFold(id.Email, activeEmail),
				IsDefault:  strings.EqualFold(id.Name, cfg.Default),
			}
		}
		jsonBytes, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Render and print identity list
	output := ui.RenderIdentityList(identities, activeEmail, cfg.Default)
	fmt.Println(output)

	return nil
}
