package cmd

import "github.com/spf13/cobra"

var canary bool

func addCanaryFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&canary, "canary", false, "Use database canary build.")
}
