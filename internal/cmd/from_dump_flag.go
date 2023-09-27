package cmd

import "github.com/spf13/cobra"

var fromDumpFlag string

func addDbFromDumpFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDumpFlag, "from-dump", "", "create the database from a local SQLite dump")
}
