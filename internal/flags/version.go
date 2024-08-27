package flags

import (
	"github.com/spf13/cobra"
)

var versionFlag string

func AddVersion(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&versionFlag, "version", "", desc)
	_ = cmd.RegisterFlagCompletionFunc("version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"latest", "canary"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func Version() string {
	return versionFlag
}
