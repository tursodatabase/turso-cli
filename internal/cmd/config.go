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
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage your CLI configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"autoupdate"}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("failed to read settings: %w", err)
		}

		switch args[0] {
		case "autoupdate":
			if args[1] != "on" && args[1] != "off" {
				return fmt.Errorf("autoupdate must be either 'on' or 'off'")
			}
			settings.SetAutoupdate(args[1])

			// trigger an update
			settings.SetLastUpdateCheck(0)
			fmt.Println("Autoupdate is now", internal.Emph(args[1]))
		default:
			return fmt.Errorf("unknown config: %s", args[0])
		}

		return nil
	},
}
