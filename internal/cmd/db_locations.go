package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/exp/maps"
)

func init() {
	dbCmd.AddCommand(regionsCmd)
}

func platformName(pl string) string {
	switch pl {
	case "fly":
		return "Fly.io Regions"
	case "aws":
		return "AWS Regions"
	case "local":
		return "Local"
	default:
		return pl
	}
}

var regionsCmd = &cobra.Command{
	Use:               "locations",
	Short:             "List available database locations.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		locations, err := mapLocations(client)
		if err != nil {
			return err
		}

		closest, err := closestLocation(client)
		if err != nil {
			return err
		}
		_, exist := locations["local"]
		if exist {
			closest = "local"
		}

		columns := make([]interface{}, 0)
		columns = append(columns, "ID↓")
		columns = append(columns, "LOCATION")
		singlePlatform := len(locations) == 1
		platforms := maps.Keys(locations)
		// Try to provide stable ordering for better UX
		// Fly comes first if present because the bottom of
		// the list is what is seen more prominently.
		priorities := map[string]int{
			"fly":   0,
			"aws":   1,
			"local": 2,
		}
		sort.Slice(platforms, func(i, j int) bool {
			iPriority, iExists := priorities[platforms[i]]
			jPriority, jExists := priorities[platforms[j]]

			if iExists && jExists {
				return iPriority < jPriority
			}

			if iExists {
				return true
			}

			if jExists {
				return false
			}
			return platforms[i] < platforms[j]
		})

		for idx, platform := range platforms {
			locs := locations[platform]

			ids := maps.Keys(locs)
			sort.Strings(ids)
			tbl := turso.LocationsTable(columns)

			if !singlePlatform {
				fmt.Println(internal.Emph(platformName(platform)))
			}
			for _, location := range ids {
				description := locs[location]
				if location == closest {
					description = fmt.Sprintf("%s  [default]", description)
					tbl.AddRow(internal.Emph(location), internal.Emph(description))
				} else {
					tbl.AddRow(location, description)
				}
			}
			tbl.Print()
			if idx != len(locations)-1 {
				fmt.Println()
			}
		}
		return nil
	},
}
