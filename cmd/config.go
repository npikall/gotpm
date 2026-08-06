package cmd

import (
	"github.com/npikall/gotpm/internal/cmds/config"
	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gotpm configuration values.",
	Long: `Manage gotpm configuration values, stored as TOML in the user config directory.

Available keys:
	- fork.path  local directory a forked package repository gets cloned into
	- fork.url   URL of the forked package repository`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set the value of a config key",
	Example: `# set the local path a fork gets cloned into
gotpm config set fork.path /home/user/typst-packages`,
	Args: cobra.ExactArgs(2), //nolint: mnd
	RunE: ConfigSetRunner,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print the value of a config key",
	Example: `# print the configured fork path
gotpm config get fork.path`,
	Args: cobra.ExactArgs(1),
	RunE: ConfigGetRunner,
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Clear the value of a config key",
	Example: `# clear the configured fork path
gotpm config unset fork.path`,
	Args: cobra.ExactArgs(1),
	RunE: ConfigUnsetRunner,
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all config keys and their values",
	Example: `gotpm config list`,
	Args:    cobra.NoArgs,
	RunE:    ConfigListRunner,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configListCmd)
}

func ConfigSetRunner(_ *cobra.Command, args []string) error {
	return config.Set(args[0], args[1])
}

func ConfigGetRunner(_ *cobra.Command, args []string) error {
	return config.Get(args[0])
}

func ConfigUnsetRunner(_ *cobra.Command, args []string) error {
	return config.Unset(args[0])
}

func ConfigListRunner(_ *cobra.Command, _ []string) error {
	return config.List()
}
