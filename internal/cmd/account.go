package cmd

import (
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "contact",
	Short: "Reach out to the makers of Turso for help or feedback",
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountBookMeetingCmd)
	accountCmd.AddCommand(accountFeedbackCmd)
}
