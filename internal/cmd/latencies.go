package cmd

import "github.com/spf13/cobra"

var latencyFlag bool

func addLatencyFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&latencyFlag, "show-latencies", "l", false, "Display latencies from your current location to each of Turso's locations")
}
