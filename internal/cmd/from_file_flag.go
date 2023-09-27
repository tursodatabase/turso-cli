package cmd

import "github.com/spf13/cobra"

var fromFileFlag string

func addDbFromFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromFileFlag, "from-file", "", "create the database from a local SQLite3-compatible file")
}
