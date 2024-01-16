package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

var quickstartCmd = &cobra.Command{
	Use:               "quickstart",
	Short:             "New to Turso? Start here!",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		fmt.Print("\nWelcome to Turso!\n\n")

		quickstartURL := "https://docs.turso.tech/quickstart"

		if err := browser.OpenURL(quickstartURL); err != nil {
			fmt.Printf("To get started with Turso, open the following URL in your browser:\n\n")
			fmt.Println(quickstartURL)
		} else {
			fmt.Println("Opening Turso Quickstart Guide in your browser...")
		}
		return nil
	},
}
