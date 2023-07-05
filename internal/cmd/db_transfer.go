package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbTransferCmd)
}

var dbTransferCmd = &cobra.Command{
	Use:               "transfer database_name org_name",
	Short:             "Transfers database to another organization",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		dbName := args[0]
		orgName := args[1]

		if _, err := getDatabase(client, dbName); err != nil {
			return err
		}

		ok, err := promptConfirmation(fmt.Sprintf("Are you sure you want to transfer database %s to %s?", dbName, orgName))
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Transfer database cancelled by the user.")
			return nil
		}

		return transfer(client, dbName, orgName)
	},
}

func transfer(client *turso.Client, dbName, orgName string) error {
	config, err := settings.ReadSettings()

	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Transfering database %s to org %s", internal.Emph(dbName), internal.Emph(orgName))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Databases.Transfer(dbName, orgName); err != nil {
		return fmt.Errorf("error transfering database")
	}

	s.Stop()
	fmt.Printf("✔  Success! Database %s transferred successfully to org %s\n", internal.Emph(dbName), internal.Emph(orgName))

	config.InvalidateDatabasesCache()

	return nil
}
