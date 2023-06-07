package cmd

import (
	"sort"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
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
		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		databases, err := client.Databases.List()
		if err != nil {
			return err
		}

		settings.SetDbNamesCache(extractDatabaseNames(databases))

		var data [][]string
		for _, database := range databases {
			data = append(data, []string{
				database.Name,
				getDatabaseRegions(database),
				getDatabaseUrl(settings, &database, false)},
			)
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] > data[j][0]
		})

		printTable([]string{"Name", "Locations", "URL"}, data)
		return nil
	},
}
