package cmd

import (
	"fmt"

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
		columns = append(columns, "MAX")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		planInfo := getPlanInfo(PlanType(userInfo.Plan))

		tbl.AddRow("storage", humanize.IBytes(usage.Total.StorageBytesUsed), planInfo.maxStorage)
		tbl.AddRow("rows read", usage.Total.RowsRead, fmt.Sprintf("%d", int(1e9)))
		tbl.AddRow("rows written", usage.Total.RowsWritten, fmt.Sprintf("%d", int(1e9)))
		tbl.AddRow("databases", usage.Total.Databases, planInfo.maxDatabases)
		tbl.AddRow("locations", usage.Total.Locations, planInfo.maxLocation)
		tbl.Print()

		return nil
	},
}
