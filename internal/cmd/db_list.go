package cmd

import (
	"context"
	"fmt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"sort"
	"time"
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		dbInfo := make(chan []string, len(databases))

		for _, database := range databases {
			database := database
			g.Go(func() error {
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
				sizeInfo, err := calculateInstancesUsedSize(instances, settings, database, token)
				if err != nil {
					size = fmt.Sprintf("fetching size failed: %s", err)
				} else {
					size = sizeInfo.PrintTotal()
				}
				dbInfo <- []string{database.Name, regions, url, size}
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return err
		}

		var data [][]string
		for range databases {
			data = append(data, <-dbInfo)
		}
		sort.Slice(data, func(i, j int) bool {
			return data[i][0] > data[j][0]
		})
		printTable([]string{"Name", "Locations", "URL", "Size"}, data)
		settings.SetDbNamesCache(extractDatabaseNames(databases))
		return nil
	},
}
