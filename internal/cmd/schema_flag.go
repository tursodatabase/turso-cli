package cmd

import "github.com/spf13/cobra"

var schemaFlag string

func addSchemaFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&schemaFlag, "schema", "", "Schema to use for the database")
	cmd.RegisterFlagCompletionFunc("schema", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}
