package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	orgCmd.AddCommand(dbTransferCmd)
}

var dbTransferCmd = &cobra.Command{
	Use:               "db-transfer <database-name> <organization-name>",
	Short:             "Transfers a database to another organization",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: dbNameAndOrgArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		dbName := args[0]
		orgName := args[1]

		if _, err := getDatabase(client, dbName, true); err != nil {
			return err
		}
		fmt.Printf("To transfer %s database to another organization, all its replicas must be updated.\n", internal.Emph(dbName))
		fmt.Printf("All your active connections to the DB will be dropped and there will be a short downtime.\n\n")

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
	invalidateDatabasesCache()

	msg := fmt.Sprintf("Transferring database %s to organization %s", internal.Emph(dbName), internal.Emph(orgName))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Databases.Transfer(dbName, orgName); err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("✔  Success! Database %s transferred successfully to organization %s\n", internal.Emph(dbName), internal.Emph(orgName))

	return nil
}
