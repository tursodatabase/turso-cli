package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	dbCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the destruction of all locations of the database.")
	addLocationFlag(destroyCmd, "Pick a database location to destroy.")
	addInstanceFlag(destroyCmd, "Pick a specific database instance to destroy.")
	destroyCmd.RegisterFlagCompletionFunc("instance", completeInstanceName)
}

var destroyCmd = &cobra.Command{
	Use:               "destroy <database-name>",
	Short:             "Destroy a database.",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		if len(args) > 1 {
			return handleDestroyMultipleDBs(args, client)
		}

		return handleDestroySingleDB(args, client)
	},
}

func handleDestroySingleDB(args []string, client *turso.Client) error {
	name := args[0]

	db, err := getDatabase(client, name)
	if err != nil {
		return nil
	}

	if instanceFlag != "" {
		if db.Group != "" {
			return fmt.Errorf("group databases do not support instance destruction.\nUse %s instead", internal.Emph("turso group locations rm "+name))
		}
		return destroyDatabaseInstance(client, name, instanceFlag)
	}

	if locationFlag != "" {
		if db.Group != "" {
			return fmt.Errorf("group databases do not support location destruction.\nUse %s instead", internal.Emph("turso group locations rm "+name+" "+locationFlag))
		}
		return destroyDatabaseRegion(client, name, locationFlag)
	}

	if yesFlag {
		return destroyDatabases(client, args)
	}

	fmt.Printf("Database %s and all its data will be destroyed.\n", internal.Emph(name))

	ok, err := promptConfirmation("Are you sure you want to do this?")
	if err != nil {
		return fmt.Errorf("could not get prompt confirmed by user: %w", err)
	}

	if !ok {
		fmt.Println("Database destruction avoided.")
		return nil
	}

	return destroyDatabases(client, args)
}

func handleDestroyMultipleDBs(args []string, client *turso.Client) error {
	if instanceFlag != "" || locationFlag != "" {
		return errors.New("can not use location nor instance flag when deleting more than 1 database")
	}

	if yesFlag {
		return destroyDatabases(client, args)
	}

	fmt.Printf("Databases %s and all their data will be destroyed.\n", internal.Emph(strings.Join(args, ", ")))

	ok, err := promptConfirmation("Are you sure you want to do this?")
	if err != nil {
		return fmt.Errorf("could not get prompt confirmed by user: %w", err)
	}

	if !ok {
		fmt.Println("Databases destruction avoided.")
		return nil
	}

	return destroyDatabases(client, args)
}
