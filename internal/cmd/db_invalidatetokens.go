package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbTokensCmd.AddCommand(dbInvalidateTokensCmd)

	dbInvalidateTokensCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the invalidation of all existing db tokens.")
}

var dbInvalidateTokensCmd = &cobra.Command{
	Use:               "invalidate database_name",
	Short:             "Rotates the keys used to create and verify database tokens making existing tokens invalid",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]

		database, err := getDatabase(client, name, true)
		if err != nil {
			return err
		}

		if yesFlag {
			return rotateAndNotify(client, database)
		}

		fmt.Printf("To invalidate %s database tokens, all its replicas must be restarted.\n", internal.Emph(name))
		fmt.Printf("All your active connections to the DB will be dropped and there will be a short downtime.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Token invalidation skipped by the user.")
			return nil
		}

		return rotateAndNotify(client, database)
	},
}

func rotateAndNotify(turso *turso.Client, database turso.Database) error {
	s := prompt.Spinner("Invalidating db auth tokens... ")
	defer s.Stop()

	if err := rotate(turso, database); err != nil {
		return err
	}

	s.Stop()
	fmt.Println("✔  Success! Tokens invalidated successfully. ")
	fmt.Printf("Run %s to get a new one!\n", internal.Emph("turso db tokens create database_name [flags]"))
	return nil
}

func rotate(turso *turso.Client, database turso.Database) error {
	if database.Group != "" {
		return turso.Groups.Rotate(database.Group)
	}
	return turso.Databases.Rotate(database.Name)
}
