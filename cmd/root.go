package cmd

import (
	"github.com/spf13/cobra"
)

// Version is the current version of gitch
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "gitch",
	Short: "A git identity manager",
	Long: `gitch helps you manage multiple git identities with ease.

Switch between work, personal, and open-source identities seamlessly.
Never commit with the wrong git identity again.

Examples:
  gitch add --name work --email work@company.com
  gitch use work
  gitch list
  gitch status`,
	Version: Version,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
