package cmd

import (
	"github.com/spf13/cobra"
)

func noSpaceArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoSpace
}

func noFilesArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}
