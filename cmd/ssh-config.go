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

func init() {
	rootCmd.AddCommand(sshConfigCmd)
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

// placeholder to avoid "declared but not used" errors during incremental build
var _ = fmt.Sprint
