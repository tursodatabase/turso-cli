package cmd

import "github.com/spf13/cobra"

var timestampFlag string

func addDbTimestampFlag(cmd *cobra.Command) {
	restoreCmd.Flags().StringVarP(&timestampFlag, "timestamp", "t", "", "UTC timestamp for the version of database at a given point in time.")
}
