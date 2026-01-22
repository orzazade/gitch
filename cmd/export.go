package cmd

import (
	"errors"
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/portability"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <file>",
	Short: "Export identities and rules to a YAML file",
	Long: `Export all gitch identities and rules to a YAML file for backup or migration.

The exported file includes:
- All identity names, emails, SSH key paths, GPG key IDs
- All auto-switch rules (directory and remote patterns)
- Export metadata (timestamp, version)

Note: SSH and GPG private keys are NOT exported, only file paths.

Examples:
  gitch export backup.yaml
  gitch export ~/gitch-backup.yaml`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config has identities
	if len(cfg.Identities) == 0 {
		fmt.Println(ui.WarningStyle.Render("Warning: No identities to export"))
		return errors.New("no identities configured")
	}

	// Export to file
	outputPath := args[0]
	if err := portability.ExportToFile(cfg, outputPath); err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	// Print success message
	identityCount := len(cfg.Identities)
	ruleCount := len(cfg.Rules)

	fmt.Println(ui.SuccessStyle.Render("Export complete!"))
	fmt.Printf("  File: %s\n", outputPath)
	fmt.Printf("  Identities: %d\n", identityCount)
	if ruleCount > 0 {
		fmt.Printf("  Rules: %d\n", ruleCount)
	}

	return nil
}
