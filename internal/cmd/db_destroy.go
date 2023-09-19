package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the destruction of all locations of the database.")
	addLocationFlag(destroyCmd, "Pick a database location to destroy.")
	addInstanceFlag(destroyCmd, "Pick a specific database instance to destroy.")
	destroyCmd.RegisterFlagCompletionFunc("instance", completeInstanceName)
}

var destroyCmd = &cobra.Command{
	Use:               "destroy database_name",
	Short:             "Destroy a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		db, err := getDatabase(client, args[0])
		if err != nil {
			return nil
		}

		name := args[0]
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
			return destroyDatabase(client, name)
		}

		fmt.Printf("Database %s, and all its data will be destroyed.\n", internal.Emph(name))

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Database destruction avoided.")
			return nil
		}

		return destroyDatabase(client, name)
	},
}
