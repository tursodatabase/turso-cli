package cmd

import "github.com/spf13/cobra"

var devPort int

func addDevPortFlag(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&devPort, "port", "p", 8080, "the port to which bind the server")
}
