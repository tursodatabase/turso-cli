package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(regionsCmd)
}

var regionsCmd = &cobra.Command{
	Use:               "locations",
	Short:             "List available database locations.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		client := createTursoClient()
		regions, err := turso.GetRegions(client)
		if err != nil {
			return err
		}
		defaultRegionId := probeClosestRegion()
		fmt.Println("ID   LOCATION")
		for idx := range regions.Ids {
			suffix := ""
			if regions.Ids[idx] == defaultRegionId {
				suffix = "  [default]"
			}
			line := fmt.Sprintf("%s  %s%s", regions.Ids[idx], regions.Descriptions[idx], suffix)
			if regions.Ids[idx] == defaultRegionId {
				line = turso.Emph(line)
			}
			fmt.Printf("%s\n", line)
		}
		settings.SetRegionsCache(regions.Ids)
		return nil
	},
}
