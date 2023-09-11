package cmd

import (
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var locationFlag string
var locationsFlag []string

func locationCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClientFromAccessToken(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	locations, _ := locations(client)
	return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp
}

func addLocationFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&locationFlag, "location", "", desc)
	cmd.RegisterFlagCompletionFunc("location", locationCompletion)
}

func addLocationsFlag(cmd *cobra.Command, desc string) {
	cmd.Flags().StringSliceVarP(&locationsFlag, "locations", "l", []string{}, desc)
	cmd.RegisterFlagCompletionFunc("locations", locationCompletion)
}
