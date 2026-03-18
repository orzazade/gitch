package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init [bash|zsh|fish]",
	Short: "Generate shell hook for automatic identity switching on directory change",
	Long: `Output shell integration code that auto-switches your git identity when you
change directories. This is the recommended way to use gitch — set it up once
and your identity follows your rules automatically.

Add the appropriate line to your shell configuration file:

Bash (~/.bashrc):
  eval "$(gitch shell-init bash)"

Zsh (~/.zshrc):
  eval "$(gitch shell-init zsh)"

Fish (~/.config/fish/config.fish):
  gitch shell-init fish | source

The hook calls "gitch autoswitch --quiet" after each directory change, so your
identity is always correct based on your configured rules.`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:                  runShellInit,
}

func init() {
	rootCmd.AddCommand(shellInitCmd)
}

func runShellInit(cmd *cobra.Command, args []string) error {
	gitch, err := os.Executable()
	if err != nil {
		gitch = "gitch"
	}

	var script string
	switch args[0] {
	case "bash":
		script = bashHook(gitch)
	case "zsh":
		script = zshHook(gitch)
	case "fish":
		script = fishHook(gitch)
	}

	fmt.Print(script)
	return nil
}

func bashHook(gitch string) string {
	return `_gitch_hook() {
  if [ -n "$GITCH_DISABLE" ]; then return; fi
  if [ "$_GITCH_PREV_DIR" = "$PWD" ]; then return; fi
  _GITCH_PREV_DIR="$PWD"
  command ` + gitch + ` autoswitch --quiet 2>/dev/null
}

if [[ ";${PROMPT_COMMAND[*]:-};" != *";_gitch_hook;"* ]]; then
  PROMPT_COMMAND=("_gitch_hook" "${PROMPT_COMMAND[@]}")
fi
`
}

func zshHook(gitch string) string {
	return `_gitch_hook() {
  if [ -n "$GITCH_DISABLE" ]; then return; fi
  command ` + gitch + ` autoswitch --quiet 2>/dev/null
}

if (( ! ${+functions[_gitch_hook]} )) || [[ "${chpwd_functions[(r)_gitch_hook]}" != "_gitch_hook" ]]; then
  chpwd_functions=(_gitch_hook $chpwd_functions)
fi
`
}

func fishHook(gitch string) string {
	return `function _gitch_hook --on-variable PWD
  if set -q GITCH_DISABLE; return; end
  command ` + gitch + ` autoswitch --quiet 2>/dev/null
end
`
}
