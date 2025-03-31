package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	dbConfigCmd.AddCommand(dbDeleteProtectionCmd)
	dbDeleteProtectionCmd.AddCommand(dbEnableDeleteProtectionCmd)
	dbDeleteProtectionCmd.AddCommand(dbDisableDeleteProtectionCmd)
	dbDeleteProtectionCmd.AddCommand(dbShowDeleteProtectionCmd)
}

var dbDeleteProtectionCmd = &cobra.Command{
	Use:               "delete-protection",
	Short:             "Manage delete-protection config of a database",
	ValidArgsFunction: noSpaceArg,
}

var dbEnableDeleteProtectionCmd = &cobra.Command{
	Use:               "enable <database-name>",
	Short:             "Disables delete protection for this database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateDeleteProtection(args[0], true)
	},
}

var dbDisableDeleteProtectionCmd = &cobra.Command{
	Use:               "disable <database-name>",
	Short:             "Disables delete protection for this database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateDeleteProtection(args[0], false)
	},
}

var dbShowDeleteProtectionCmd = &cobra.Command{
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
		fmt.Print(deleteProtectionMessage(config.IsDeleteProtected()))
		return err
	},
}

func updateDeleteProtection(name string, deleteProtection bool) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	database, err := getDatabase(client, name, true)
	if err != nil {
		return err
	}
	return client.Databases.UpdateConfig(database.Name, turso.DatabaseConfig{DeleteProtection: &deleteProtection})
}

func deleteProtectionMessage(status bool) string {
	msg := "off"
	if status {
		msg = "on"
	}
	return fmt.Sprintf("Delete Protection %s\n", internal.Emph(msg))
}
