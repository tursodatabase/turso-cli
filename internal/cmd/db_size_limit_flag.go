package cmd

import "github.com/spf13/cobra"

var sizeLimitFlag string

func addSizeLimitFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&sizeLimitFlag, "size-limit", "", "The maximum size of the database in bytes. Values with units are accepted, e.g. 1mb, 256mb, 1gb")
}
