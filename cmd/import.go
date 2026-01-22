package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/portability"
	"github.com/orzazade/gitch/internal/ssh"
	"github.com/orzazade/gitch/internal/ui"
	"github.com/spf13/cobra"
)

var importForce bool

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import identities and rules from a YAML file",
	Long: `Import gitch identities and rules from a YAML file.

When importing, if an identity or rule already exists:
- You will be prompted to overwrite, skip, or abort
- Use --force to overwrite all conflicts without prompting

Note: SSH key files must exist at the referenced paths for SSH features to work.

Examples:
  gitch import backup.yaml
  gitch import ~/gitch-backup.yaml --force`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.Flags().BoolVarP(&importForce, "force", "f", false, "Overwrite all conflicts without prompting")
}

func runImport(cmd *cobra.Command, args []string) error {
	// Load current config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Parse import file
	inputPath := args[0]
	export, err := portability.ImportFromFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Validate imported identities and warn about missing SSH keys
	for _, id := range export.Identities {
		if err := id.Validate(); err != nil {
			return fmt.Errorf("invalid identity %q in import file: %w", id.Name, err)
		}

		// Warn if SSH key path doesn't exist (but continue import)
		if id.SSHKeyPath != "" {
			expanded, err := ssh.ExpandPath(id.SSHKeyPath)
			if err == nil {
				if _, statErr := os.Stat(expanded); os.IsNotExist(statErr) {
					fmt.Fprintf(os.Stderr, "Warning: SSH key not found: %s (identity: %s)\n", id.SSHKeyPath, id.Name)
				}
			}
		}
	}

	// Detect conflicts
	conflicts := portability.DetectConflicts(cfg, export)

	// Build overwrite map
	overwrite := make(map[string]bool)

	if len(conflicts) > 0 {
		if importForce {
			// Force mode: overwrite all conflicts
			for _, c := range conflicts {
				overwrite[c.Key] = true
			}
		} else {
			// Interactive mode: prompt for each conflict
			reader := bufio.NewReader(os.Stdin)
			for _, c := range conflicts {
				shouldOverwrite, abort, err := promptConflict(reader, c)
				if err != nil {
					return fmt.Errorf("failed to read input: %w", err)
				}
				if abort {
					fmt.Println(ui.WarningStyle.Render("Import aborted"))
					return nil
				}
				overwrite[c.Key] = shouldOverwrite
			}
		}
	}

	// Merge config
	result, err := portability.MergeConfig(cfg, export, overwrite)
	if err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}

	// Handle default identity from import
	if export.Default != "" && cfg.Default == "" {
		// Check if the default identity exists in the merged config
		if _, err := cfg.GetIdentity(export.Default); err == nil {
			cfg.Default = export.Default
		}
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Print summary
	printImportSummary(inputPath, result)

	return nil
}

func promptConflict(reader *bufio.Reader, c portability.Conflict) (overwrite bool, abort bool, err error) {
	switch c.Type {
	case portability.IdentityConflict:
		existing := c.Existing.(config.Identity)
		incoming := c.Incoming.(config.Identity)

		fmt.Println()
		fmt.Printf("Identity %q already exists:\n", c.Key)
		fmt.Printf("  Existing: %s", existing.Email)
		if existing.SSHKeyPath != "" {
			fmt.Printf(" (SSH: %s)", existing.SSHKeyPath)
		}
		if existing.GPGKeyID != "" {
			fmt.Printf(" (GPG: %s)", existing.GPGKeyID)
		}
		fmt.Println()

		fmt.Printf("  Incoming: %s", incoming.Email)
		if incoming.SSHKeyPath != "" {
			fmt.Printf(" (SSH: %s)", incoming.SSHKeyPath)
		}
		if incoming.GPGKeyID != "" {
			fmt.Printf(" (GPG: %s)", incoming.GPGKeyID)
		}
		fmt.Println()

	case portability.RuleConflict:
		fmt.Println()
		fmt.Printf("Rule %q already exists with different identity\n", c.Key)
	}

	fmt.Print("  [o]verwrite / [s]kip / [a]bort? ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "o", "overwrite":
		return true, false, nil
	case "a", "abort":
		return false, true, nil
	default:
		// Default to skip
		return false, false, nil
	}
}

func printImportSummary(path string, result *portability.ImportResult) {
	fmt.Println()
	fmt.Println(ui.SuccessStyle.Render("Import complete!"))
	fmt.Printf("  File: %s\n", path)

	hasOutput := false

	if len(result.AddedIdentities) > 0 {
		fmt.Printf("  + %d identities added\n", len(result.AddedIdentities))
		hasOutput = true
	}
	if len(result.UpdatedIdentities) > 0 {
		fmt.Printf("  ~ %d identities updated\n", len(result.UpdatedIdentities))
		hasOutput = true
	}
	if len(result.AddedRules) > 0 {
		fmt.Printf("  + %d rules added\n", len(result.AddedRules))
		hasOutput = true
	}
	if len(result.UpdatedRules) > 0 {
		fmt.Printf("  ~ %d rules updated\n", len(result.UpdatedRules))
		hasOutput = true
	}

	// Count skipped identities and rules separately
	skippedIdentities := 0
	skippedRules := 0
	for _, s := range result.Skipped {
		if strings.HasPrefix(s, "identity:") {
			skippedIdentities++
		} else if strings.HasPrefix(s, "rule:") {
			skippedRules++
		}
	}

	if skippedIdentities > 0 {
		fmt.Printf("  - %d identities skipped\n", skippedIdentities)
		hasOutput = true
	}
	if skippedRules > 0 {
		fmt.Printf("  - %d rules skipped\n", skippedRules)
		hasOutput = true
	}

	if !hasOutput {
		fmt.Println("  No changes (config already up to date)")
	}
}
