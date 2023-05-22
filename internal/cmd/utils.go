package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/sync/errgroup"
)

func createTursoClientFromAccessToken(warnMultipleAccessTokenSources bool) (*turso.Client, error) {
	token, err := getAccessToken(warnMultipleAccessTokenSources)
	if err != nil {
		return nil, err
	}
	return tursoClient(token)
}

func createUnauthenticatedTursoClient() (*turso.Client, error) {
	return tursoClient("")
}

func tursoClient(token string) (*turso.Client, error) {
	urlStr := getTursoUrl()
	tursoUrl, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("error creating turso client: could not parse turso URL %s: %w", urlStr, err)
	}

	config, err := settings.ReadSettings()
	if err != nil {
		return nil, fmt.Errorf("error creating turso client: could not parse turso URL %s: %w", urlStr, err)
	}

	org := config.Organization()
	return turso.New(tursoUrl, token, version, org), nil
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

func getDatabaseUrl(settings *settings.Settings, db *turso.Database, password bool) string {
	return getUrl(settings, db, nil, "libsql")
}

func getInstanceUrl(settings *settings.Settings, db *turso.Database, inst *turso.Instance) string {
	return getUrl(settings, db, inst, "libsql")
}

func getDatabaseHttpUrl(settings *settings.Settings, db *turso.Database) string {
	return getUrl(settings, db, nil, "https")
}

func getInstanceHttpUrl(settings *settings.Settings, db *turso.Database, inst *turso.Instance) string {
	return getUrl(settings, db, inst, "https")
}

func getUrl(settings *settings.Settings, db *turso.Database, inst *turso.Instance, scheme string) string {
	host := db.Hostname
	if inst != nil {
		host = inst.Hostname
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func getDatabaseRegions(db turso.Database) string {
	regions := make([]string, 0, len(db.Regions))
	for _, region := range db.Regions {
		if region == db.PrimaryRegion {
			region = fmt.Sprintf("%s (primary)", internal.Emph(region))
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

func destroyDatabase(client *turso.Client, name string) error {
	start := time.Now()
	s := prompt.Spinner(fmt.Sprintf("Destroying database %s... ", internal.Emph(name)))
	defer s.Stop()

	if err := client.Databases.Delete(name); err != nil {
		return err
	}
	s.Stop()
	elapsed := time.Since(start)

	fmt.Printf("Destroyed database %s in %d seconds.\n", internal.Emph(name), int(elapsed.Seconds()))
	settings, err := settings.ReadSettings()
	if err == nil {
		settings.InvalidateDbNamesCache()
	}

	return nil
}

func destroyDatabaseRegion(client *turso.Client, database, region string) error {
	if !isValidLocation(client, region) {
		return fmt.Errorf("location '%s' is not a valid one", region)
	}

	s := prompt.Spinner(fmt.Sprintf("Destroying region %s of database %s... ", internal.Emph(region), internal.Emph(database)))
	defer s.Stop()

	db, err := getDatabase(client, database)
	if err != nil {
		return err
	}

	instances, err := client.Instances.List(db.Name)
	if err != nil {
		return err
	}

	instances = filterInstancesByRegion(instances, region)
	if len(instances) == 0 {
		return fmt.Errorf("could not find any instances of database %s in location %s", db.Name, region)
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

	s.Stop()
	fmt.Printf("Destroyed %d instances in location %s of database %s.\n", len(replicas), internal.Emph(region), internal.Emph(db.Name))
	if primary != nil {
		destroyAllCmd := fmt.Sprintf("turso db destroy %s", database)
		fmt.Printf("Primary was not destroyed. To destroy it, with the whole database, run '%s'\n", destroyAllCmd)
	}

	return nil
}

func destroyDatabaseInstance(client *turso.Client, database, instance string) error {
	s := prompt.Spinner(fmt.Sprintf("Destroying instance %s of database %s... ", instance, internal.Emph(database)))
	defer s.Stop()

	err := deleteDatabaseInstance(client, database, instance)
	if err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("Destroyed instance %s of database %s.\n", internal.Emph(instance), internal.Emph(database))
	return nil
}

func deleteDatabaseInstance(client *turso.Client, database, instance string) error {
	err := client.Instances.Delete(database, instance)
	if err != nil {
		if err.Error() == "could not find database "+database+" to delete instance from" {
			return fmt.Errorf("database %s not found. List known databases using %s", internal.Emph(database), internal.Emph("turso db list"))
		}
		if err.Error() == "could not find instance "+instance+" of database "+database {
			return fmt.Errorf("instance %s not found for database %s. List known instances using %s", internal.Emph(instance), internal.Emph(database), internal.Emph("turso db show "+database))
		}
		return err
	}
	return nil
}

func getTursoUrl() string {
	host := os.Getenv("TURSO_API_BASEURL")
	if host == "" {
		host = "https://api.turso.io"
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

func dbNameArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClientFromAccessToken(false)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 0 {
		return getDatabaseNames(client), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

func isSQLiteFile(file *os.File) (bool, error) {
	defer file.Seek(0, io.SeekStart)
	header := make([]byte, 16)
	_, err := file.Read(header)
	if err != nil && err != io.EOF {
		return false, err
	}

	if string(header) == "SQLite format 3\000" {
		return true, nil
	}

	return false, nil
}

func fetchLatestVersion() (string, error) {
	client, err := createUnauthenticatedTursoClient()
	if err != nil {
		return "", err
	}
	resp, err := client.Get("/releases/latest", nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error getting latest release: %s", resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var versionResp struct {
		Version string `json:"latest"`
	}
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return "", err
	}
	if len(versionResp.Version) == 0 {
		return "", fmt.Errorf("got empty version for latest release")
	}
	return versionResp.Version, nil
}
