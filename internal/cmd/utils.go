package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
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

func startSpinner(text string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 40*time.Millisecond)
	s.Prefix = text
	s.Start()
	return s
}

func destroyDatabase(name string) error {
	start := time.Now()
	s := startSpinner(fmt.Sprintf("Destroying database %s... ", emph(name)))
	if err := turso.Databases.Delete(name); err != nil {
		return err
	}
	s.Stop()
	elapsed := time.Since(start)

	fmt.Printf("Destroyed database %s in %d seconds.\n", emph(name), int(elapsed.Seconds()))
	settings, err := settings.ReadSettings()
	if err == nil {
		settings.InvalidateDbNamesCache()
	}

	settings.DeleteDatabase(name)
	return nil
}

func destroyDatabaseRegion(database, region string) error {
	db, err := getDatabase(database)
	if err != nil {
		return err
	}

	if db.Type != "logical" {
		return fmt.Errorf("database '%s' does not support the destroy operation with region argument", db.Name)
	}

	instances, err := turso.Instances.List(db.Name)
	if err != nil {
		return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
	}

	instance := findInstanceFromRegion(instances, region)
	if instance == nil {
		return fmt.Errorf("could not find any instance of database %s on region %s", db.Name, region)
	}

	err = turso.Instances.Delete(db.Name, instance.Name)
	if err != nil {
		// TODO: remove this once wait stopped bug is fixed
		time.Sleep(3 * time.Second)
		err = turso.Instances.Delete(db.Name, instance.Name)
		if err != nil {
			return fmt.Errorf("could not delete instance %s from region %s: %w", instance.Name, region, err)
		}
	}

	fmt.Printf("Destroyed instance %s in region %s of database %s.\n", emph(instance.Name), emph(region), emph(db.Name))
	return nil
}
