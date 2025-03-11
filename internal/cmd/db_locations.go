package cmd

import (
	"fmt"
	"sort"
	"strings"

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

		ids := maps.Keys(locations)

		awsIds := make([]string, 0, len(ids))
		flyIds := make([]string, 0, len(ids))

		for _, id := range ids {
			if strings.HasPrefix(id, "aws-") {
				awsIds = append(awsIds, id)
			} else {
				flyIds = append(flyIds, id)
			}
		}

		sort.Strings(awsIds)
		sort.Strings(flyIds)

		columns = append(columns, "ID↓")
		columns = append(columns, "LOCATION")

		flyTbl := turso.LocationsTable(columns)
		awsTbl := turso.LocationsTable(columns)

		if len(flyIds) > 0 {
			fmt.Println(internal.Emph("Fly.io Regions:"))
			for _, location := range flyIds {
				description := locations[location]
				if location == closest {
					description = fmt.Sprintf("%s  [default]", description)
					flyTbl.AddRow(internal.Emph(location), internal.Emph(description))
				} else {
					flyTbl.AddRow(location, description)
				}
			}
			flyTbl.Print()
			fmt.Println("")
		}

		fmt.Println(internal.Emph("AWS (beta) Regions:"))
		for _, location := range awsIds {
			description := locations[location]
			if location == closest {
				description = fmt.Sprintf("%s  [default]", description)
				awsTbl.AddRow(internal.Emph(location), internal.Emph(description))
			} else {
				awsTbl.AddRow(location, description)
			}
		}
		awsTbl.Print()
		return nil
	},
}
