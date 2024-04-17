package cmd

import "github.com/spf13/cobra"

var typeFlag string

func addTypeFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&typeFlag, "type", "regular", "Type of the database to create. Possible values: regular, schema.")
	cmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"regular", "schema"}, cobra.ShellCompDirectiveNoFileComp
	})
}
