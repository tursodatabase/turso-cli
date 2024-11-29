package cmd

import (
	"github.com/spf13/cobra"
)

var betaFlag bool

func addBetaFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&betaFlag, "beta", false, "enable beta features")
}
