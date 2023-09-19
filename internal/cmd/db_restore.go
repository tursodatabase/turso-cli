package cmd

import (
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the restoration of all locations of the database.")
	addDbTimestampFlag(restoreCmd)
}

var restoreCmd = &cobra.Command{
	Use:               "restore database_name 2023-09-25T14:53:00",
	Short:             "Restore a database to a given point in time.",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]
		timestamp, err := time.Parse("2006-01-02T15:04:05", args[1])
		if err != nil {
			return fmt.Errorf("could not parse timestamp using 'yyyy-MM-ddThh:mm:ss' pattern: %w", err)
		}

		if yesFlag {
			return restoreDatabase(client, name, timestamp)
		}

		fmt.Printf("Database %s, all its replicas, and data will be replaced.\n", internal.Emph(name))

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Database restoration cancelled.")
			return nil
		}

		return restoreDatabase(client, name, timestamp)
	},
}
