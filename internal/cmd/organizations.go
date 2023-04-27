package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(orgsCmd)
}

var orgsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Manage your organizations",
}
