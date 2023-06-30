package cmd

import (
	"github.com/spf13/cobra"
)

var instanceFlag string

func addInstanceFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&instanceFlag, "instance", "", desc)
}
