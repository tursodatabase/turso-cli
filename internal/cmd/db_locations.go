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

		var ids []string
		ids = maps.Keys(locations)
		sort.Strings(ids)
		columns = append(columns, "ID↓")
		columns = append(columns, "LOCATION")

		tbl := turso.LocationsTable(columns)

		for _, location := range ids {
			description := locations[location]
			if location == closest {
				description = fmt.Sprintf("%s  [default]", description)
				tbl.AddRow(internal.Emph(location), internal.Emph(description))
			} else {
				tbl.AddRow(location, description)
			}
		}
		tbl.Print()
		return nil
	},
}
