package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configSetCmd.AddCommand(configSetAutoUpdateCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage your CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration value",
}

var configSetAutoUpdateCmd = &cobra.Command{
	Use:   "autoupdate <on|off>",
	Short: "Configure autoupdate behavior",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{"on", "off"}, cobra.ShellCompDirectiveNoFileComp
		}
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		value := args[0]
		if value != "on" && value != "off" {
			return fmt.Errorf("autoupdate must be either 'on' or 'off'")
		}

		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("failed to read settings: %w", err)
		}

		settings.SetAutoupdate(value)
		settings.SetLastUpdateCheck(0) // trigger an update
		fmt.Println("Autoupdate set to", internal.Emph(value))

		return nil
	},
}
