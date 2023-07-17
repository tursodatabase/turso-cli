package cmd

import (
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "contact",
	Short: "Reach out for help or feedback for the makers of Turso",
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountBookMeetingCmd)
	accountCmd.AddCommand(accountFeedbackCmd)
}
