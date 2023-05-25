package cmd

import "github.com/spf13/cobra"

var canaryFlag bool

func addCanaryFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&canaryFlag, "canary", false, "Use database canary build.")
}
