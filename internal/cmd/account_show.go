package cmd

import (
	"fmt"
	"strconv"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/dustin/go-humanize"
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
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		userInfo, err := client.Users.GetUser()
		if err != nil {
			return err
		}

		usage, err := client.Organizations.Usage()
		if err != nil {
			return err
		}

		fmt.Printf("You are currently on %s plan.\n", internal.Emph(userInfo.Plan))
		fmt.Println()

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "LIMIT")
		columns = append(columns, "PERCENTAGE")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		planInfo := getPlanInfo(PlanType(userInfo.Plan))

		maxStorage, err := humanize.ParseBytes(planInfo.maxStorage)
		if err != nil {
			return err
		}
		maxDatabases, err := strconv.ParseUint(planInfo.maxDatabases, 10, 64)
		if err != nil {
			return err
		}
		maxLocations, err := strconv.ParseUint(planInfo.maxLocation, 10, 64)
		if err != nil {
			return err
		}
		addResourceRowBytes(tbl, "storage", usage.Total.StorageBytesUsed, maxStorage)
		addResourceRowCount(tbl, "rows read", usage.Total.RowsRead, uint64(1e9))
		addResourceRowCount(tbl, "rows written", usage.Total.RowsWritten, uint64(25*1e6))
		addResourceRowCount(tbl, "databases", usage.Total.Databases, maxDatabases)
		addResourceRowCount(tbl, "locations", usage.Total.Locations, maxLocations)
		tbl.Print()

		return nil
	},
}

func addResourceRowBytes(tbl table.Table, resource string, used, limit uint64) {
	tbl.AddRow(resource, humanize.IBytes(used), humanize.IBytes(limit), percentage(float64(used), float64(limit)))
}

func addResourceRowCount(tbl table.Table, resource string, used, limit uint64) {
	tbl.AddRow(resource, used, limit, percentage(float64(used), float64(limit)))
}

func percentage(used, limit float64) string {
	return fmt.Sprintf("%.0f %%", used/limit*100)
}
