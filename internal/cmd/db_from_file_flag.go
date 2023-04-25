package cmd

import "github.com/spf13/cobra"

var dbFromFile string

func addDbFromFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&dbFromFile, "from-file", "", "create the database from a local SQLite3-compatible file")
}
