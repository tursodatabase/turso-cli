package cmd

import (
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var locationFlag string

func addLocationFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&locationFlag, "location", "", desc)
	cmd.RegisterFlagCompletionFunc("location", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := authedTursoClient()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		locations, _ := locations(client)
		return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp
	})
}
