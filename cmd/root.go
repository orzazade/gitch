package cmd

import (
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the current version of gitch
var Version = "0.1.0"

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

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// XDG config path: ~/.config/gitch/config.yaml
	configPath, err := xdg.ConfigFile("gitch/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine config path: %v\n", err)
		return
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("identities", []interface{}{})
	viper.SetDefault("default", "")

	// Read config (ignore ConfigFileNotFoundError - first run is normal)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Real error, not just missing file
			fmt.Fprintf(os.Stderr, "Warning: error reading config: %v\n", err)
		}
	}
}
