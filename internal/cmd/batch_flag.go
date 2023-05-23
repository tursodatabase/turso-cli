package cmd

import "github.com/spf13/cobra"

var batchFlag int

func addBatchFlag(cmd *cobra.Command, usage string) {
	cmd.Flags().IntVar(&batchFlag, "batch", 1000, usage)
}
