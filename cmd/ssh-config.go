package cmd

import (
	"fmt"

	"github.com/orzazade/gitch/internal/config"
	"github.com/orzazade/gitch/internal/ssh"
	"github.com/spf13/cobra"
)

var sshConfigCmd = &cobra.Command{
	Use:   "ssh-config",
	Short: "Manage SSH config Host aliases for identities",
	Long: `Generate and manage SSH config Host blocks for your identities.

gitch can generate SSH config Host aliases that allow you to use different
SSH keys for different GitHub/GitLab accounts. Each identity with an SSH key
will get Host aliases like "github-<name>" and "gitlab-<name>".

This enables you to clone repositories using the identity-specific host alias:
  git clone git@github-work:company/repo.git

Commands:
  generate    Print SSH config Host blocks to stdout
  update      Write Host blocks to ~/.ssh/config with backup

Examples:
  gitch ssh-config generate
  gitch ssh-config update
  gitch ssh-config update --dry-run`,
}

var sshConfigGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print SSH config Host blocks for all identities with SSH keys",
	Long: `Generate SSH config Host blocks for all identities that have SSH keys.

The output can be manually added to ~/.ssh/config, or you can use the
'update' command to automatically apply the changes with a backup.

Each identity with an SSH key gets Host aliases for github.com and gitlab.com,
allowing you to use different SSH keys for different accounts.

Example output:
  Host github-work
      HostName github.com
      User git
      IdentityFile ~/.ssh/id_ed25519_work
      IdentitiesOnly yes`,
	RunE: runSSHConfigGenerate,
}

func init() {
	rootCmd.AddCommand(sshConfigCmd)
	sshConfigCmd.AddCommand(sshConfigGenerateCmd)
}

// collectHosts gathers HostConfigs from all identities with SSH keys
func collectHosts(cfg *config.Config) []ssh.HostConfig {
	var hosts []ssh.HostConfig
	identities := cfg.ListIdentities()
	for _, identity := range identities {
		identityHosts := ssh.IdentityToHosts(identity)
		if identityHosts != nil {
			hosts = append(hosts, identityHosts...)
		}
	}
	return hosts
}

func runSSHConfigGenerate(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Collect hosts from identities
	hosts := collectHosts(cfg)

	if len(hosts) == 0 {
		fmt.Println("No identities with SSH keys found.")
		return nil
	}

	// Generate the config block
	block := ssh.GenerateConfigBlock(hosts)

	// Print the block
	fmt.Print(block)

	// Print hint
	fmt.Println("\n# To apply, run: gitch ssh-config update")

	return nil
}
