package cmd

import (
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var groupFlag string

func addGroupFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&groupFlag, "group", "", "create the database in the specified group")
	cmd.RegisterFlagCompletionFunc("group", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := createTursoClientFromAccessToken(false)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groups, _ := groupNames(client)
		return groups, cobra.ShellCompDirectiveNoFileComp
	})
}

var fromDBFlag string
var timestampFlag string

func addFromDBFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDBFlag, "from-db", "", "Select another database to copy data from. To use data from a past version of the selected database, see the 'timestamp' flag.")
	cmd.RegisterFlagCompletionFunc("from-db", dbNameArg)
	cmd.Flags().StringVar(&timestampFlag, "timestamp", "", "Set a point in time in the past to copy data from the selected database. Must be used with the 'from-db' flag. Must be in the format 'yyyy-MM-ddThh:mm::ss'")
}

func parseTimestampFlag() (*time.Time, error) {
	if timestampFlag == "" {
		return nil, nil
	}
	if fromDBFlag == "" {
		return nil, fmt.Errorf("--timestamp cannot be used without specifying --from-db")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampFlag)
	if err != nil {
		return nil, fmt.Errorf("provided timestamp was not in 'yyyy-MM-ddThh:mm::ss' format")
	}
	return &timestamp, nil
}

func parseDBSeedFlags() (*turso.DBSeed, error) {
	timestamp, err := parseTimestampFlag()
	if err != nil {
		return nil, err
	}

	if fromDBFlag != "" {
		return &turso.DBSeed{
			Type:      "database",
			Name:      fromDBFlag,
			Timestamp: timestamp,
		}, nil
	}

	return nil, nil
}
