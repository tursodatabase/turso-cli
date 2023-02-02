package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func createTursoClient() *turso.Client {
	tursoUrl, err := url.Parse(getTursoUrl())
	if err != nil {
		log.Fatal(fmt.Errorf("error creating turso client: could not parse turso URL %s: %w", getTursoUrl(), err))
	}

	token, err := getAccessToken()
	if err != nil {
		log.Fatal(fmt.Errorf("error creating Turso client: %w", err))
	}

	return turso.New(tursoUrl, token)
}

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

func filterInstancesByRegion(instances []turso.Instance, region string) []turso.Instance {
	result := []turso.Instance{}
	for _, instance := range instances {
		if instance.Region == region {
			result = append(result, instance)
		}
	}
	return result
}

func extractPrimary(instances []turso.Instance) (primary *turso.Instance, others []turso.Instance) {
	result := []turso.Instance{}
	for _, instance := range instances {
		if instance.Type == "primary" {
			primary = &instance
			continue
		}
		result = append(result, instance)
	}
	return primary, result
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

func getInstanceUrl(settings *settings.Settings, db turso.Database, inst turso.Instance) string {
	dbSettings := settings.GetDatabaseSettings(db.ID)
	if dbSettings == nil {
		// Backwards compatibility with old settings files.
		dbSettings = settings.GetDatabaseSettings(db.Name)
	}

	url := "<n/a>"
	if dbSettings != nil {
		url = fmt.Sprintf("https://%s:%s@%s", dbSettings.Username, dbSettings.Password, inst.Hostname)
	}
	return url
}

func getDatabaseRegions(db turso.Database) string {
	if db.Type != "logical" {
		return db.Region
	}

	regions := make([]string, 0, len(db.Regions))
	for _, region := range db.Regions {
		if region == db.PrimaryRegion {
			region = fmt.Sprintf("%s (primary)", emph(region))
		}
		regions = append(regions, region)
	}

	return strings.Join(regions, ", ")
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

func printTable(header []string, data [][]string) {
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

func startLoadingBar(text string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[36], 800*time.Millisecond)
	s.Prefix = text
	s.Start()
	return s
}

func destroyDatabase(client *turso.Client, name string) error {
	start := time.Now()
	s := startSpinner(fmt.Sprintf("Destroying database %s... ", emph(name)))
	defer s.Stop()
	if err := client.Databases.Delete(name); err != nil {
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

func destroyDatabaseReplicas(client *turso.Client, database, region string) error {
	db, err := getDatabase(client, database)
	if err != nil {
		return err
	}

	if db.Type != "logical" {
		return fmt.Errorf("database '%s' does not support the destroy operation with region argument", db.Name)
	}

	instances, err := client.Instances.List(db.Name)
	if err != nil {
		return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
	}

	instances = filterInstancesByRegion(instances, region)
	if len(instances) == 0 {
		return fmt.Errorf("could not find any instances of database %s on region %s", db.Name, region)
	}

	primary, replicas := extractPrimary(instances)
	g := errgroup.Group{}
	for i := range replicas {
		replica := replicas[i]
		g.Go(func() error { return destroyDatabaseInstance(client, db.Name, replica.Name) })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	fmt.Printf("Destroyed %d instances in region %s of database %s.\n", len(replicas), emph(region), emph(db.Name))
	if primary != nil {
		destroyAllCmd := fmt.Sprintf("turso db destroy %s --all", database)
		return fmt.Errorf("Primary was not destroyed. To destroy it, with the whole database, run '%s'\n", destroyAllCmd)
	}

	return nil
}

func destroyDatabaseInstance(client *turso.Client, database, instance string) error {
	if err := client.Instances.Delete(database, instance); err != nil {
		// TODO: remove this once wait stopped bug is fixed
		time.Sleep(3 * time.Second)
		err = client.Instances.Delete(database, instance)
		if err != nil {
			return fmt.Errorf("could not delete instance %s: %w", instance, err)
		}
	}
	return nil
}

func getTursoUrl() string {
	host := os.Getenv("TURSO_API_BASEURL")
	if host == "" {
		host = "https://api.chiseledge.com"
	}
	return host
}
