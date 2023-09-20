package cmd

import "github.com/spf13/cobra"

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

	cmd.Flags().StringVar(&timestampFlag, "timestamp", "", "When used with --from-db option, fork will represent state of the origin database at given point in time")
	cmd.Flags().MarkHidden("timestamp")
}
