package cmd

import "github.com/spf13/cobra"

var verboseFlag bool

func addVerboseFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&verboseFlag, "verbose", false, "Show detailed information")
}
