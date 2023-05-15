package cmd

import (
	"fmt"

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

		data := [][]string{}
		for _, database := range databases {
			url := getDatabaseUrl(settings, &database, false)
			regions := getDatabaseRegions(database)

			instances, err := client.Instances.List(database.Name)
			if err != nil {
				return err
			}

			token, err := client.Databases.Token(database.Name, "1d", true)
			if err != nil {
				return err
			}
			var size string
			sizeInfo, err := calculateInstancesUsedSize(client, instances, settings, database, token)
			if err != nil {
				size = fmt.Sprintf("fetching size failed: %s", err)
			} else {
				size = sizeInfo.PrintTotal()
			}
			data = append(data, []string{database.Name, regions, url, size})
		}
		printTable([]string{"Name", "Locations", "URL", "Size"}, data)
		settings.SetDbNamesCache(extractDatabaseNames(databases))
		return nil
	},
}
