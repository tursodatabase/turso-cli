package cmd

import (
	"fmt"
	"sort"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
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
		client, err := createTursoClientFromAccessToken(true)
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

		ids := maps.Keys(locations)
		sort.Strings(ids)

		fmt.Println("ID   LOCATION")
		for _, location := range ids {
			description := locations[location]

			suffix := ""
			if location == closest {
				suffix = "  [default]"
			}

			line := fmt.Sprintf("%s  %s%s", location, description, suffix)
			if location == closest {
				line = internal.Emph(line)
			}

			fmt.Printf("%s\n", line)
		}
		return nil
	},
}
