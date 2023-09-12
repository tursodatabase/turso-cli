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

var forkFlag string

func addForkFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&forkFlag, "fork", "", "fork the new database from an existing one")
	cmd.Flags().MarkHidden("fork")
	cmd.RegisterFlagCompletionFunc("fork", dbNameArg)
}
