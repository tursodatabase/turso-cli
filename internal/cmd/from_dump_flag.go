package cmd

import "github.com/spf13/cobra"

var fromDumpFlag string
var fromDumpURLFlag string

func addDbFromDumpFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDumpFlag, "from-dump", "", "create the database from a local SQLite dump")
}

func addDbFromDumpURLFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDumpURLFlag, "from-dump-url", "", "create the database from a remote SQLite dump")
}
