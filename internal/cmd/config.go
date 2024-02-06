package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetCmd)
	configSetCmd.AddCommand(configSetAutoUpdateCmd)
	configSetCmd.AddCommand(configSetTokenCmd)

	configCmd.AddCommand(configCacheCmd)
	configCacheCmd.AddCommand(configCacheClearCmd)
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

var configSetTokenCmd = &cobra.Command{
	Use:   "token <jwt>",
	Short: "Configure the token used by turso",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		config, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("failed to read settings: %w", err)
		}

		token := args[0]
		if !isJwtTokenValid(token) {
			return fmt.Errorf("invalid token")
		}

		config.SetToken(token)
		if err := settings.TryToPersistChanges(); err != nil {
			return fmt.Errorf("%w\nIf the issue persists, set your token to the %s environment variable instead", err, internal.Emph(ENV_ACCESS_TOKEN))
		}
		fmt.Println("Token set succesfully.")
		return nil
	},
}

var configCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage your CLI cache",
}

var configCacheClearCmd = &cobra.Command{
	Use:               "clear",
	Short:             "Clear your CLI local cache",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := settings.ClearCache(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		fmt.Println("Local cache cleared successfully")
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:               "path",
	Short:             "Get the path to the CLI configuration file",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = settings.ReadSettings()
		fmt.Println(settings.Path())
		return nil
	},
}
