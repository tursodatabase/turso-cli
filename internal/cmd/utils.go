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

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/olekukonko/tablewriter"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/sync/errgroup"
)

const (
	tursoDefaultBaseURL = "https://api.turso.tech"
)

func authedTursoClient() (*turso.Client, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}
	return tursoClient(token)
}

func unauthedTursoClient() (*turso.Client, error) {
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
		return nil, fmt.Errorf("error creating turso client: could not read settings file: %w", err)
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

func getDatabaseUrl(db *turso.Database) string {
	return getUrl(db, nil, "libsql")
}

func getInstanceUrl(db *turso.Database, inst *turso.Instance) string {
	return getUrl(db, inst, "libsql")
}

func getDatabaseHttpUrl(db *turso.Database) string {
	return getUrl(db, nil, "https")
}

func getUrl(db *turso.Database, inst *turso.Instance, scheme string) string {
	host := db.Hostname
	if inst != nil {
		host = inst.Hostname
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func formatBool(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func getDatabaseLocations(db turso.Database) string {
	return formatLocations(db.Regions, db.PrimaryRegion)
}

func formatLocations(locations []string, primary string) string {
	formatted := make([]string, 0, len(locations))
	for _, location := range locations {
		if location == primary {
			location = fmt.Sprintf("%s (primary)", internal.Emph(location))
		}
		formatted = append(formatted, location)
	}

	return strings.Join(formatted, ", ")
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

func destroyDatabases(client *turso.Client, names []string) error {
	if len(names) == 0 {
		return nil
	}

	invalidateDatabasesCache()
	invalidateGroupsCache(client.Org)
	invalidateDbTokenCache()
	settings.PersistChanges()

	var g errgroup.Group

	msg := "Destroying databases..."
	if len(names) == 1 {
		msg = fmt.Sprintf("Destroying database %s... ", internal.Emph(names[0]))
	}
	s := prompt.Spinner(msg)

	start := time.Now()
	for _, name := range names {
		name := name
		g.Go(func() error {
			return client.Databases.Delete(name)
		})
	}

	if err := g.Wait(); err != nil {
		s.Stop()
		return err
	}

	s.Stop()

	elapsed := time.Since(start)

	msg = fmt.Sprintf("Destroyed %d databases in %d seconds.\n", len(names), int(elapsed.Seconds()))
	if len(names) == 1 {
		msg = fmt.Sprintf("Destroyed database %s in %s.\n", internal.Emph(names[0]), elapsed.Round(time.Millisecond).String())
	}
	fmt.Println(msg)

	return nil
}

func destroyDatabaseRegion(client *turso.Client, database, region string) error {
	if !isValidLocation(client, region) {
		return fmt.Errorf("location '%s' is not a valid one", region)
	}

	s := prompt.Spinner(fmt.Sprintf("Destroying location %s of database %s... ", internal.Emph(region), internal.Emph(database)))
	defer s.Stop()

	db, err := getDatabase(client, database, true)
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
	config, _ := settings.ReadSettings() // ok to ignore, we'll fallback to default
	url := config.GetBaseURL()
	if url == "" {
		url = tursoDefaultBaseURL
	}
	return url
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
	client, err := authedTursoClient()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 0 {
		return getDatabaseNames(client), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

func dbNameAndOrgArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 1 {
		orgs, _ := client.Organizations.List()
		return extractOrgNames(orgs), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	return dbNameArg(cmd, args, toComplete)
}

func fetchLatestVersion() (string, error) {
	client, err := unauthedTursoClient()
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

func instancesAndUsage(client *turso.Client, database string) (instances []turso.Instance, usage turso.DbUsage, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		instances, err = client.Instances.List(database)
		return
	})
	g.Go(func() (err error) {
		usage, err = client.Databases.Usage(database)
		return
	})
	err = g.Wait()
	return
}

func isInteractive() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
