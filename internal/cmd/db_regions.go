package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var regionsCmd = &cobra.Command{
	Use:               "regions",
	Short:             "List available database regions.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := createTursoClient()
		fmt.Println("ID   LOCATION")
		regions, err := turso.GetRegions(client)
		if err != nil {
			return err
		}
		defaultRegionId := probeClosestRegion()
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
		return nil
	},
}
