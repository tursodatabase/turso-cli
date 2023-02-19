package cmd

import (
	"bufio"
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
	"golang.org/x/sync/errgroup"
)

func createTursoClient() *turso.Client {
	token, err := getAccessToken()
	if err != nil {
		log.Fatal(fmt.Errorf("error creating Turso client: %w", err))
	}

	return tursoClient(&token)
}

func createUnauthenticatedTursoClient() *turso.Client {
	return tursoClient(nil)
}

func tursoClient(token *string) *turso.Client {
	tursoUrl, err := url.Parse(getTursoUrl())
	if err != nil {
		log.Fatal(fmt.Errorf("error creating turso client: could not parse turso URL %s: %w", getTursoUrl(), err))
	}

	return turso.New(tursoUrl, token)
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
			region = fmt.Sprintf("%s (primary)", turso.Emph(region))
		}
		regions = append(regions, region)
	}

	return strings.Join(regions, ", ")
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

func startLoadingBar(text string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[36], 800*time.Millisecond)
	s.Suffix = "\n" + text
	s.Start()
	return s
}

func destroyDatabase(client *turso.Client, name string) error {
	start := time.Now()
	s := startLoadingBar(fmt.Sprintf("Destroying database %s... ", turso.Emph(name)))
	defer s.Stop()
	if err := client.Databases.Delete(name); err != nil {
		return err
	}
	s.Stop()
	elapsed := time.Since(start)

	fmt.Printf("Destroyed database %s in %d seconds.\n", turso.Emph(name), int(elapsed.Seconds()))
	settings, err := settings.ReadSettings()
	if err == nil {
		settings.InvalidateDbNamesCache()
	}

	settings.DeleteDatabase(name)
	return nil
}

func destroyDatabaseRegion(client *turso.Client, database, region string) error {
	if !isValidRegion(client, region) {
		return fmt.Errorf("region '%s' is not a valid one", region)
	}

	db, err := getDatabase(client, database)
	if err != nil {
		return err
	}

	if db.Type != "logical" {
		return fmt.Errorf("database '%s' does not support the destroy operation with region argument", db.Name)
	}

	instances, err := client.Instances.List(db.Name)
	if err != nil {
		return err
	}

	instances = filterInstancesByRegion(instances, region)
	if len(instances) == 0 {
		return fmt.Errorf("could not find any instances of database %s on region %s", db.Name, region)
	}

	primary, replicas := extractPrimary(instances)
	g := errgroup.Group{}
	for i := range replicas {
		replica := replicas[i]
		g.Go(func() error { return deleteDatabaseInstance(client, db.Name, replica.Name) })
	}

	if err := g.Wait(); err != nil {
		return err
	}

	fmt.Printf("Destroyed %d instances in region %s of database %s.\n", len(replicas), turso.Emph(region), turso.Emph(db.Name))
	if primary != nil {
		destroyAllCmd := fmt.Sprintf("turso db destroy %s", database)
		fmt.Printf("Primary was not destroyed. To destroy it, with the whole database, run '%s'\n", destroyAllCmd)
	}

	return nil
}

func destroyDatabaseInstance(client *turso.Client, database, instance string) error {
	err := deleteDatabaseInstance(client, database, instance)
	if err != nil {
		return err
	}
	fmt.Printf("Destroyed instance %s of database %s.\n", turso.Emph(instance), turso.Emph(database))
	return nil
}

func deleteDatabaseInstance(client *turso.Client, database, instance string) error {
	err := client.Instances.Delete(database, instance)
	if err != nil {
		return err
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

func promptConfirmation(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)

	for i := 0; i < 3; i++ {
		fmt.Printf("%s [y/n]: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		input = strings.ToLower(strings.TrimSpace(input))
		if input == "y" || input == "yes" {
			return true, nil
		} else if input == "n" || input == "no" {
			return false, nil
		}

		fmt.Println("Please answer with yes or no.")
	}

	return false, fmt.Errorf("could not get prompt confirmed by user")
}
