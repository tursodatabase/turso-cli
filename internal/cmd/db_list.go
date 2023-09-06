package cmd

import (
	"fmt"
	"sort"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:               "list",
	Short:             "List databases.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		databases, err := client.Databases.List()
		if err != nil {
			return err
		}
		setDatabasesCache(databases)
		var data [][]string
		var helps []string
		for _, database := range databases {
			row := []string{database.Name, getDatabaseLocations(database), getDatabaseUrl(&database)}
			if len(database.Regions) == 0 {
				help := fmt.Sprintf("🛠 Run %s to finish your database creation!", internal.Emph("turso db replicate "+database.Name))
				helps = append(helps, help)
			}
			data = append(data, row)
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] > data[j][0]
		})

		printTable([]string{"Name", "Locations", "URL"}, data)

		if len(helps) > 0 {
			fmt.Println()
			for _, help := range helps {
				fmt.Println(help)
			}
		}
		return nil
	},
}
