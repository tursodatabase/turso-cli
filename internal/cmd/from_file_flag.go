package cmd

import "github.com/spf13/cobra"

var fromFileFlag string

func addFromFileFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().StringVarP(&fromFileFlag, "from-file", "f", "", usage)
}
