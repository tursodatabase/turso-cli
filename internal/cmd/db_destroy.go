package cmd

import (
	"fmt"
	"strings"

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
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		if instanceFlag != "" || locationFlag != "" {
			for _, name := range args {
				db, err := getDatabase(client, name)
				if err != nil {
					return nil
				}

				if instanceFlag != "" {
					if db.Group != "" {
						return fmt.Errorf("group databases do not support instance destruction.\nUse %s instead", internal.Emph("turso group locations rm "+name))
					}

					if err := destroyDatabaseInstance(client, name, instanceFlag); err != nil {
						return err
					}
				}

				if locationFlag != "" {
					if db.Group != "" {
						return fmt.Errorf("group databases do not support location destruction.\nUse %s instead", internal.Emph("turso group locations rm "+name+" "+locationFlag))
					}

					if err := destroyDatabaseRegion(client, name, locationFlag); err != nil {
						return err
					}
				}
			}
		}

		if yesFlag {
			return destroyDatabases(client, args)
		}

		if len(args) > 1 {
			fmt.Printf("Databases %s and all their data will be destroyed.\n", internal.Emph(strings.Join(args, ", ")))
		} else {
			fmt.Printf("Database %s, and all its data will be destroyed.\n", internal.Emph(args[0]))
		}

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Database destruction avoided.")
			return nil
		}

		return destroyDatabases(client, args)
	},
}
