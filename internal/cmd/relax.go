package cmd

import (
	"github.com/chiselstrike/turso-cli/internal/tetris"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(funCmd)
}

var funCmd = &cobra.Command{
	Use:               "relax",
	Short:             "Sometimes you feel like you're working too hard... relax!",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		tetris.Start()
		return nil
	},
}
