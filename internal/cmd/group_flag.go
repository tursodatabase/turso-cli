package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

var groupFlag string

func addGroupFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&groupFlag, "group", "", "create the database in the specified group")
	cmd.Flags().MarkHidden("group")
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
	cmd.Flags().StringVar(&fromDBFlag, "from-db", "", "Creates the new database based on an existing one")
	cmd.Flags().MarkHidden("from-db")
	cmd.RegisterFlagCompletionFunc("from-db", dbNameArg)

	cmd.Flags().StringVar(&timestampFlag, "timestamp", "", "When used with --from-db option, new database will represent state of its origin at given point in time")
	cmd.Flags().MarkHidden("timestamp")
}

func parseTimestampFlag() (*time.Time, error) {
	var timestamp *time.Time
	if fromDBFlag != "" {
		if timestampFlag != "" {
			ts, err := time.Parse("2006-01-02T15:04:05", timestampFlag)
			if err != nil {
				return nil, fmt.Errorf("provided timestamp was not in 'yyyy-MM-ddThh:mm::ss' format")
			}
			timestamp = &ts
		}
	} else if timestampFlag != "" {
		return nil, fmt.Errorf("--timestamp cannot be used without specifying --from-db option")
	}
	return timestamp, nil
}
