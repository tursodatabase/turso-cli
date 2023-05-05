package cmd

import "github.com/spf13/cobra"

var waitFlag bool

func addWaitFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().BoolVarP(&waitFlag, "wait", "w", false, desc)
}
