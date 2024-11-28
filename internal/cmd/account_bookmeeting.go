package cmd

import (
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var accountBookMeetingCmd = &cobra.Command{
	Use:               "bookmeeting",
	Short:             "Book a meeting with the Turso team.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return browser.OpenURL("https://tur.so/cli-chat")
	},
}
