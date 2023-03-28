package cmd

import (
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
		client, err := createTursoClient()
		if err != nil {
			return err
		}
		databases, err := getDatabases(client)
		if err != nil {
			return err
		}
		data := [][]string{}
		for _, database := range databases {
			url := getDatabaseUrl(settings, &database, false)
			regions := getDatabaseRegions(database)
			data = append(data, []string{database.Name, regions, url})
		}
		printTable([]string{"Name", "Locations", "URL"}, data)
		settings.SetDbNamesCache(extractDatabaseNames(databases))
		return nil
	},
}
