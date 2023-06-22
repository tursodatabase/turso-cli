package cmd

import (
	"github.com/spf13/cobra"
)

var outputFlag string

func addOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&outputFlag, "output", "o", "", "Set output file")
}
