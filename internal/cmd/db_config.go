package cmd

import "github.com/spf13/cobra"

func init() {
	dbCmd.AddCommand(dbConfigCmd)
}

var dbConfigCmd = &cobra.Command{
	Use:               "config",
	Short:             "Manage db config",
	ValidArgsFunction: noSpaceArg,
}
