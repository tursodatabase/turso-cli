package flags

import (
	"github.com/spf13/cobra"
)

var extensionsFlag string

func AddExtensions(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&extensionsFlag, "extensions", "", desc)
	_ = cmd.RegisterFlagCompletionFunc("extensions", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"all", "none"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func Extensions() string {
	return extensionsFlag
}
