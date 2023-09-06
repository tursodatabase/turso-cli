package cmd

import (
	"github.com/spf13/cobra"
)

var yesFlag bool

func addYesFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, desc)
}
