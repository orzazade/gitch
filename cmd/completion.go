package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for gitch.

To load completions:

Bash:
  # Linux:
  $ gitch completion bash > /etc/bash_completion.d/gitch

  # macOS with Homebrew:
  $ gitch completion bash > $(brew --prefix)/etc/bash_completion.d/gitch

  # Current session only:
  $ source <(gitch completion bash)

Zsh:
  # If shell completion is not already enabled:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Add to fpath (run once):
  $ gitch completion zsh > "${fpath[1]}/_gitch"

  # Or for Oh My Zsh:
  $ gitch completion zsh > ~/.oh-my-zsh/completions/_gitch

  # Current session only:
  $ source <(gitch completion zsh)

Fish:
  $ gitch completion fish > ~/.config/fish/completions/gitch.fish

  # Current session only:
  $ gitch completion fish | source
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
