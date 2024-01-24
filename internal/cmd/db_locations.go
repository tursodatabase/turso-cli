package cmd

import (
	"fmt"
	"math"
	"sort"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/exp/maps"
)

func init() {
	dbCmd.AddCommand(regionsCmd)
	addLatencyFlag(regionsCmd)
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
		locations, err := locations(client)
		if err != nil {
			return err
		}

		closest, err := closestLocation(client)
		if err != nil {
			return err
		}

		columns := make([]interface{}, 0)

		lats := make(map[string]int)
		var ids []string
		if latencyFlag {
			lats, err = latencies(client)
			if err != nil {
				return err
			}
			ids = maps.Keys(lats)
			sort.Slice(ids, func(i, j int) bool {
				return lats[ids[i]] < lats[ids[j]]
			})
			columns = append(columns, "ID")
			columns = append(columns, "LOCATION")
			columns = append(columns, "LATENCY↓")
		} else {
			ids = maps.Keys(locations)
			sort.Strings(ids)
			columns = append(columns, "ID↓")
			columns = append(columns, "LOCATION")
		}

		tbl := turso.LocationsTable(columns)

		for _, location := range ids {
			description := locations[location]
			lat, ok := lats[location]
			var latency string
			if ok && lat != math.MaxInt {
				latency = fmt.Sprintf("%dms", lat)
			} else {
				latency = "???"
			}

			if location == closest {
				description = fmt.Sprintf("%s  [default]", description)
				if latencyFlag {
					tbl.AddRow(internal.Emph(location), internal.Emph(description), internal.Emph(latency))
				} else {
					tbl.AddRow(internal.Emph(location), internal.Emph(description))
				}
			} else {
				if latencyFlag {
					tbl.AddRow(location, description, latency)
				} else {
					tbl.AddRow(location, description)
				}
			}
		}
		tbl.Print()
		return nil
	},
}
