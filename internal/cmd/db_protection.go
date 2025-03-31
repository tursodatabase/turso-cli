package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	dbCmd.AddCommand(dbProtectionCmd)
	dbProtectionCmd.AddCommand(dbProtectionEnableCmd)
	dbProtectionCmd.AddCommand(dbProtectionDisableCmd)
	dbProtectionCmd.AddCommand(dbProtectionShowCmd)
}

var dbProtectionCmd = &cobra.Command{
	Use:               "protection",
	Short:             "Manage delete protection of a database",
	ValidArgsFunction: noSpaceArg,
}

var dbProtectionEnableCmd = &cobra.Command{
	Use:               "enable <database-name>",
	Short:             "Enable delete protection for a database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return setDatabaseProtection(args[0], true)
	},
}

var dbProtectionDisableCmd = &cobra.Command{
	Use:               "disable <database-name>",
	Short:             "Disable delete protection for a database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return setDatabaseProtection(args[0], false)
	},
}

var dbProtectionShowCmd = &cobra.Command{
	Use:               "show <database-name>",
	Short:             "Shows the delete protection status of a database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		name := args[0]

		database, err := getDatabase(client, name, true)
		if err != nil {
			return err
		}
		config, err := client.Databases.GetConfig(database.Name)
		if err != nil {
			return err
		}
		fmt.Print(protectionMessage(config.IsDeleteProtected()))
		return nil
	},
}

func setDatabaseProtection(name string, protect bool) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	database, err := getDatabase(client, name, true)
	if err != nil {
		return err
	}

	err = client.Databases.UpdateConfig(database.Name, turso.DatabaseConfig{DeleteProtection: &protect})
	if err != nil {
		action := "enable"
		if !protect {
			action = "disable"
		}
		return fmt.Errorf("failed to %s delete protection for database %s: %w", action, name, err)
	}

	fmt.Print(protectionMessage(protect))
	return nil
}

func protectionMessage(protected bool) string {
	status := "disabled"
	if protected {
		status = "enabled"
	}
	return fmt.Sprintf("Delete protection %s\n", internal.Emph(status))
}
