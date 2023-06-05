package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"golang.org/x/sync/errgroup"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/fatih/color"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

var accountShowCmd = &cobra.Command{
	Use:               "show",
	Short:             "Show your current account plan.",
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

		userInfo, err := client.Users.GetUser()
		if err != nil {
			return err
		}

		numDatabases := len(databases)
		numLocations := 0
		inspectRet := InspectInfo{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		// FIXME: this should be done at the server so we can enforce it
		var dbTokens []string
		var dbInstances [][]turso.Instance
		var instanceCount int
		for _, database := range databases {
			numLocations += len(database.Regions)
			instances, err := client.Instances.List(database.Name)
			if err != nil {
				return err
			}

			token, err := client.Databases.Token(database.Name, "1d", true)
			if err != nil {
				return err
			}

			dbTokens = append(dbTokens, token)
			instanceCount += len(instances)
			dbInstances = append(dbInstances, instances)
		}
		inspectResCh := make(chan *InspectInfo, instanceCount)
		g, ctx := errgroup.WithContext(ctx)
		for idx, database := range databases {
			idx := idx
			database := database
			for _, instance := range dbInstances[idx] {
				instance := instance
				g.Go(func() error {
					url := getInstanceHttpUrl(settings, &database, &instance)
					ret, err := inspectInstance(ctx, url, dbTokens[idx], instance.Region, false)
					if err != nil {
						return err
					}
					inspectResCh <- ret
					return nil
				})
			}
		}
		if err := g.Wait(); err != nil {
			return err
		}
		for i := 0; i < instanceCount; i++ {
			ret := <-inspectResCh
			inspectRet.Accumulate(ret)
		}

		fmt.Printf("You are currently on %s plan.\n", internal.Emph(userInfo.Plan))
		fmt.Println()

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "MAX")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		planInfo := getPlanInfo(PlanType(userInfo.Plan))

		tbl.AddRow("storage", inspectRet.PrintTotalStorage(), planInfo.maxStorage)
		tbl.AddRow("rows read", inspectRet.TotalRowsReadCount(), fmt.Sprintf("%d", int(1e9)))
		tbl.AddRow("databases", numDatabases, planInfo.maxDatabases)
		tbl.AddRow("locations", numLocations, planInfo.maxLocation)
		tbl.Print()

		return nil
	},
}
