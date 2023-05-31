package cmd

import (
	"fmt"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var showUrlFlag bool
var showBasicAuthFlag bool
var showHttpUrlFlag bool
var showInstanceUrlsFlag bool
var showInstanceUrlFlag string
var passwordFlag string
var yesFlag bool
var instanceFlag string

func getInstanceNames(client *turso.Client, dbName string) []string {
	instances, err := client.Instances.List(dbName)
	if err != nil {
		return nil
	}
	result := []string{}
	for _, instance := range instances {
		result = append(result, instance.Name)
	}
	return result
}

func extractDatabaseNames(databases []turso.Database) []string {
	names := make([]string, 0)
	for _, database := range databases {
		names = append(names, database.Name)
	}
	return names
}

func fetchDatabaseNames(client *turso.Client) []string {
	databases, err := client.Databases.List()
	if err != nil {
		return []string{}
	}
	return extractDatabaseNames(databases)
}

func getDatabase(client *turso.Client, name string) (turso.Database, error) {
	databases, err := client.Databases.List()
	if err != nil {
		return turso.Database{}, err
	}

	for _, database := range databases {
		if database.Name == name {
			return database, nil
		}
	}

	return turso.Database{}, fmt.Errorf("database %s not found. List known databases using %s", internal.Emph(name), internal.Emph("turso db list"))
}

func getDatabaseNames(client *turso.Client) []string {
	settings, err := settings.ReadSettings()
	if err != nil {
		return fetchDatabaseNames(client)
	}
	cached_names := settings.GetDbNamesCache()
	if cached_names != nil {
		return cached_names
	}
	names := fetchDatabaseNames(client)
	settings.SetDbNamesCache(names)
	return names
}

func completeInstanceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClientFromAccessToken(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		return getInstanceNames(client, args[0]), cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(dbCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

const ENV_ACCESS_TOKEN = "TURSO_API_TOKEN"

func getAccessToken(warnMultipleAccessTokenSources bool) (string, error) {
	envToken := os.Getenv(ENV_ACCESS_TOKEN)
	settings, err := settings.ReadSettings()
	if err != nil {
		return "", fmt.Errorf("could not read local settings")
	}
	settingsToken := settings.GetToken()

	if !noMultipleTokenSourcesWarning && envToken != "" && settingsToken != "" && warnMultipleAccessTokenSources {
		fmt.Printf("Warning: User logged in as %s but TURSO_API_TOKEN environment variable is set so proceeding to use it instead\n\n", settings.GetUsername())
	}
	if envToken != "" {
		return envToken, nil
	}
	if !isJwtTokenValid(settingsToken) {
		return "", fmt.Errorf("user not logged in, please login with %s", internal.Emph("turso auth login"))
	}

	return settingsToken, nil
}

func getHost() string {
	host := os.Getenv("TURSO_API_BASEURL")
	if host == "" {
		host = "https://api.turso.io"
	}
	return host
}

func locations(client *turso.Client) (map[string]string, error) {
	settings, _ := settings.ReadSettings()
	if locations := settings.LocationsCache(); locations != nil {
		return locations, nil
	}

	locations, err := client.Locations.Get()
	if err != nil {
		return nil, err
	}

	settings.SetLocationsCache(locations)
	return locations, nil
}

func closestLocation(client *turso.Client) (string, error) {
	settings, _ := settings.ReadSettings()
	if closest := settings.ClosestLocationCache(); closest != "" {
		return closest, nil
	}

	closest, err := client.Locations.Closest()
	if err != nil {
		// We fallback to ams if we are unable to probe the closest location.
		return "ams", err
	}

	settings.SetClosestLocationCache(closest)
	return closest, nil
}

func isNorthAmerica(location string) bool {
	switch location {
	case "atl", "bos", "den", "dfw", "ewr", "iad", "lax", "mia", "ord", "sea", "sjc", "yul", "yyz":
		return true
	default:
		return false
	}
}

func suggestedLocation(mainLocation string, locations map[string]string) (string, string) {
	if isNorthAmerica(mainLocation) {
		return "fra", "Europe"
	} else {
		return "iad", "North America"
	}
}

func isValidLocation(client *turso.Client, location string) bool {
	locations, err := locations(client)
	if err != nil {
		return true
	}
	_, ok := locations[location]
	return ok
}

func locationDescription(client *turso.Client, id string) string {
	locations, _ := locations(client)
	if desc, ok := locations[id]; ok {
		return desc
	}
	return fmt.Sprintf("Location ID: %s", id)
}
