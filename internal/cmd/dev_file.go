package cmd

import "github.com/spf13/cobra"

var devFile string

func addDevFileFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&devFile, "db-file", "f", "", "A file name to persist the data of this dev session")
}
