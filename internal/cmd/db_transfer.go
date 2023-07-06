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
	orgCmd.AddCommand(dbTransferCmd)
}

var dbTransferCmd = &cobra.Command{
	Use:               "transfer database_name org_name",
	Short:             "Transfers database to another organization",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: dbNameAndOrgArgs,
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

		ok, err := promptConfirmation(fmt.Sprintf("Are you sure you want to transfer database %s to organization %s?", internal.Emph(dbName), internal.Emph(orgName)))
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
	config.InvalidateDatabasesCache()

	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Transferring database %s to organization %s", internal.Emph(dbName), internal.Emph(orgName))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Databases.Transfer(dbName, orgName); err != nil {
		return fmt.Errorf("error transferring database: %w", err)
	}

	s.Stop()
	fmt.Printf("✔  Success! Database %s transferred successfully to organization %s\n", internal.Emph(dbName), internal.Emph(orgName))

	return nil
}
