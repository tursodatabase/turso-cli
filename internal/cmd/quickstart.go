package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

func init() {
	rootCmd.AddCommand(quickstartCmd)
	rootCmd.AddCommand(postInstallCmd)
}

var quickstartCmd = &cobra.Command{
	Use:               "quickstart",
	Short:             "New to Turso? Start here!",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		quickstart(false)
		return nil
	},
}

var postInstallCmd = &cobra.Command{
	Use:    "post-install",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if checkSignedIn() {
			return nil
		}
		quickstart(true)
		return nil
	},
}

func quickstart(headless bool) {
	fmt.Print("\nWelcome to Turso!\n\n")

	quickstartURL := "https://docs.turso.tech/quickstart"
	if headless || browser.OpenURL(quickstartURL) != nil {
		fmt.Printf("To get started with Turso, open the following URL in your browser:\n\n")
		fmt.Println(quickstartURL)
		return
	}

	fmt.Println("Opening Turso Quickstart Guide in your browser...")
}

func checkSignedIn() bool {
	settings, err := settings.ReadSettings()
	if err != nil {
		return false
	}

	return isJwtTokenValid(settings.GetToken())
}
