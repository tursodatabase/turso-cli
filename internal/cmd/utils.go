package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func dbNameValidator(argIndex int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return nil
	}
}

func regionArgValidator(argIndex int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		region := args[argIndex]
		for _, v := range regionIds {
			if v == region {
				return nil
			}
		}
		return fmt.Errorf("there is no %s region available", region)
	}
}

func findInstanceFromRegion(instances []turso.Instance, region string) *turso.Instance {
	for _, instance := range instances {
		if instance.Region == region {
			return &instance
		}
	}
	return nil
}

func getDatabaseUrl(settings *settings.Settings, db turso.Database) string {
	dbSettings := settings.GetDatabaseSettings(db.ID)
	if dbSettings == nil {
		// Backwards compatibility with old settings files.
		dbSettings = settings.GetDatabaseSettings(db.Name)
	}

	url := "<n/a>"
	if dbSettings != nil {
		url = dbSettings.GetURL()
	}
	return url
}

func displayRegions(instances []turso.Instance) string {
	regions := make(map[string]bool)
	for _, instance := range instances {
		region := instance.Region
		regions[region] = regions[region] || (instance.Type == "primary")
	}

	list := make([]string, 0)
	for region, primary := range regions {
		tag := region
		if primary {
			tag = emph(region) + " (primary)"
		}
		list = append(list, tag)
	}

	return strings.Join(list, ", ")
}

func printTable(title string, header []string, data [][]string) {
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader(header)
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoFormatHeaders(true)

	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("  ")
	table.SetNoWhiteSpace(true)
	table.SetTablePadding("     ")

	table.AppendBulk(data)

	table.Render()
}
