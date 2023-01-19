package cmd

import (
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Turso",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the platform.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return browser.OpenURL("https://api.chiseledge.com")
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
}
