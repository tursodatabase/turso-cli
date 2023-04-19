package cmd

import "github.com/spf13/cobra"

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage your account plan and billing",
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountBookMeetingCmd)
}
