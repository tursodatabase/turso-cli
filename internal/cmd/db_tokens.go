package cmd

import "github.com/spf13/cobra"

func init() {
	dbCmd.AddCommand(dbTokensCmd)
}

var dbTokensCmd = &cobra.Command{
	Use:               "tokens",
	Short:             "Manage db tokens",
	ValidArgsFunction: noSpaceArg,
}
