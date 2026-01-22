package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/orzazade/gitch/internal/prompt"
)

var initCmd = &cobra.Command{
	Use:   "init [bash|zsh|fish]",
	Short: "Print shell integration code for prompt",
	Long: `Print shell integration code that shows your current git identity in the prompt.

Add the output to your shell configuration file:

Bash (~/.bashrc):
  eval "$(gitch init bash)"

Zsh (~/.zshrc):
  eval "$(gitch init zsh)"

Fish (~/.config/fish/config.fish):
  gitch init fish | source

After adding, restart your shell or source the config file.
The prompt will show your current identity like: [work] $`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	shell := args[0]

	// Detect prompt framework and add compatibility note if found
	framework := prompt.DetectPromptFramework()
	if framework != prompt.FrameworkNone {
		fmt.Printf("# Note: Detected %s. This integration should work alongside it.\n", framework.String())
		fmt.Println("# If you experience issues, see: gitch help init")
		fmt.Println()
	}

	// Output shell-specific init code
	switch shell {
	case "bash":
		fmt.Print(prompt.BashInit())
	case "zsh":
		fmt.Print(prompt.ZshInit())
	case "fish":
		fmt.Print(prompt.FishInit())
	}

	return nil
}
