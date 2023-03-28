package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
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

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		databases, err := getDatabases(client)
		if err != nil {
			return err
		}

		numDatabases := len(databases)
		numLocations := 0
		inspectRet := InspectInfo{}
		// FIXME: this should be done at the server so we can enforce it
		for _, database := range databases {
			numLocations += len(database.Regions)
			instances, err := client.Instances.List(database.Name)
			if err != nil {
				return err
			}
			for _, instance := range instances {
				url := getInstanceHttpUrl(settings, &database, &instance)
				ret, err := inspect(url, instance.Region, false)
				if err != nil {
					return err
				}
				inspectRet.Accumulate(ret)
			}
		}

		fmt.Printf("You are currently on %s plan.\n", turso.Emph("starter"))
		fmt.Println()

		headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
		columnFmt := color.New(color.FgBlue).SprintfFunc()

		columns := make([]interface{}, 0)
		columns = append(columns, "resource")
		columns = append(columns, "used")
		columns = append(columns, "max")

		tbl := table.New(columns...)
		tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

		tbl.AddRow("storage", inspectRet.PrintTotal(), "8GB")
		tbl.AddRow("databases", numDatabases, "3")
		tbl.AddRow("locations", numLocations, "3")
		tbl.Print()

		return nil
	},
}
