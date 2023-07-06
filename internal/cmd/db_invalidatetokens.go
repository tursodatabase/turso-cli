package cmd

import (
	"fmt"

	"github.com/chiselstrike/turso-cli/internal"
	"github.com/chiselstrike/turso-cli/internal/prompt"
	"github.com/chiselstrike/turso-cli/internal/turso"
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

		if _, err := getDatabase(client, name); err != nil {
			return err
		}

		if yesFlag {
			return rotate(client, name)
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

		return rotate(client, name)
	},
}

func rotate(turso *turso.Client, name string) error {
	s := prompt.Spinner("Invalidating db auth tokens... ")
	defer s.Stop()

	if err := turso.Databases.Rotate(name); err != nil {
		s.Stop()
		return fmt.Errorf("your database does not support tokens")
	}

	s.Stop()
	fmt.Println("✔  Success! Tokens invalidated successfully. ")
	fmt.Printf("Run %s to get a new one!\n", internal.Emph("turso db tokens create database_name [flags]"))
	return nil
}
