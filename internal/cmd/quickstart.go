package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

var quickstartCmd = &cobra.Command{
	Use:               "quickstart",
	Short:             "Turso quick quickstart.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fmt.Print("\nWelcome to Turso!\n\n")
		fmt.Printf("If you are a new user, please sign up with %s; otherwise login\n", turso.Emph("turso auth signup"))
		fmt.Printf("with %s. When you are authenticated, you can create a new\n", turso.Emph("turso auth login"))
		fmt.Printf("database with %s. You can also run %s for help.\n", turso.Emph("turso db create"), turso.Emph("turso help"))
		fmt.Printf("\nFor a more comprehensive getting started guide, open the following URL:\n\n")
		fmt.Printf("  https://docs.turso.tech/tutorials/get-started-turso-cli\n\n")
		return nil
	},
}
