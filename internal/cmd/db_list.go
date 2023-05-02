package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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
		databases, err := getDatabases(client)
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

			inspectRet := InspectInfo{}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			g, ctx := errgroup.WithContext(ctx)
			results := make(chan *InspectInfo, len(instances))
			for _, instance := range instances {
				loopInstance := instance
				g.Go(func() error {
					url := getInstanceHttpUrl(settings, &database, &loopInstance)
					ret, err := inspect(ctx, url, token, loopInstance.Region, verboseFlag)
					if err != nil {
						return err
					}
					results <- ret
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("timeout while inspecting database. It's possible that this database is too old and does not support inspecting or one of the instances is not reachable")
				}
				return err
			}
			for range instances {
				ret := <-results
				inspectRet.Accumulate(ret)
			}

			data = append(data, []string{database.Name, regions, url, inspectRet.PrintTotal()})
		}
		printTable([]string{"Name", "Locations", "URL", "Size"}, data)
		settings.SetDbNamesCache(extractDatabaseNames(databases))
		return nil
	},
}
