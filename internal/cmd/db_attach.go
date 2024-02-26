package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	dbConfigCmd.AddCommand(dbAttachCmd)
	dbAttachCmd.AddCommand(dbEnableAttachCmd)
	dbAttachCmd.AddCommand(dbDisableAttachCmd)
	dbAttachCmd.AddCommand(dbShowAttachStatusCmd)
}

var dbAttachCmd = &cobra.Command{
	Use:               "attach",
	Short:             "Manage attach config of a database",
	ValidArgsFunction: noSpaceArg,
}

var dbEnableAttachCmd = &cobra.Command{
	Use:               "allow <database-name>",
	Short:             "Allows this database to be attached by other databases",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateAttachStatus(args[0], true)
	},
}

var dbDisableAttachCmd = &cobra.Command{
	Use:               "disallow <database-name>",
	Short:             "Disallows this database to be attached by other databases",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateAttachStatus(args[0], false)
	},
}

var dbShowAttachStatusCmd = &cobra.Command{
	Use:               "show <database-name>",
	Short:             "Shows the attach status of a database",
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
		fmt.Print(attachMessage(config.AllowAttach))
		return err
	},
}

func updateAttachStatus(name string, allowAttach bool) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	database, err := getDatabase(client, name, true)
	if err != nil {
		return err
	}
	return client.Databases.UpdateConfig(database.Name, turso.DatabaseConfig{AllowAttach: allowAttach})
}

func attachMessage(attach bool) string {
	status := "not allowed"
	if attach {
		status = "allowed"
	}
	return fmt.Sprintf("Attach %s\n", internal.Emph(status))
}
