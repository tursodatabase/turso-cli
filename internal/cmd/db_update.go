package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbUpdateCmd)
	dbUpdateCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the update of the database.")
}

var dbUpdateCmd = &cobra.Command{
	Use:               "update database_name",
	Short:             "Updates the database to the latest turso version",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]

		if _, err := getDatabase(client, name); err != nil {
			return err
		}

		if yesFlag {
			return update(client, name)
		}

		fmt.Printf("To update %s database, all its replicas must be updated.\n", internal.Emph(name))
		fmt.Printf("All your active connections to the DB will be dropped and there will be a short downtime.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Database update skipped by the user.")
			return nil
		}

		return update(client, name)
	},
}

func update(client *turso.Client, name string) error {
	msg := fmt.Sprintf("Updating database %s", internal.Emph(name))
	s := prompt.Spinner(msg)

	if err := client.Databases.Update(name); err != nil {
		s.Stop()
		return fmt.Errorf("error updating database")
	}

	s.Stop()
	fmt.Printf("✔  Success! Database %s updated successfully\n", internal.Emph(name))
	return nil
}
