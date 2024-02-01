package cmd

import "github.com/spf13/cobra"

var queriesFlag bool

func addQueriesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&queriesFlag, "queries", false, "Display database queries statistics")
}
