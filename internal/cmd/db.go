package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var (
	showUrlFlag          bool
	showHttpUrlFlag      bool
	showInstanceUrlsFlag bool
	showInstanceUrlFlag  string
	showBranchesFlag     bool
)

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

func getDatabase(client *turso.Client, name string, fresh ...bool) (turso.Database, error) {
	if len(fresh) > 0 && fresh[0] {
		database, err := client.Databases.Get(name)
		if err != nil {
			return turso.Database{}, err
		}
		updateDatabaseCache(map[string]turso.Database{database.Name: database})
		return database, nil
	}

	databases, err := getDatabases(client, fresh...)
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

func getDatabases(client *turso.Client, fresh ...bool) ([]turso.Database, error) {
	skipCache := len(fresh) > 0 && fresh[0]
	if cachedNames := getDatabasesCache(); !skipCache && cachedNames != nil {
		return cachedNames, nil
	}
	r, err := client.Databases.List(turso.DatabaseListOptions{})
	if err != nil {
		return nil, err
	}
	setDatabasesCache(r.Databases)
	return r.Databases, nil
}

func getDatabasesMap(client *turso.Client, fresh bool) (map[string]turso.Database, error) {
	databases, err := getDatabases(client, fresh)
	if err != nil {
		return nil, err
	}
	databasesMap := make(map[string]turso.Database)
	for _, db := range databases {
		databasesMap[db.Name] = db
	}
	return databasesMap, nil
}

func getDatabaseNames(client *turso.Client) []string {
	databases, err := getDatabases(client)
	if err != nil {
		return []string{}
	}
	return extractDatabaseNames(databases)
}

func completeInstanceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := authedTursoClient()
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

var ErrNotLoggedIn = fmt.Errorf("user not logged in, please login with %s", internal.Emph("turso auth login"))

func getAccessToken() (string, error) {
	token, err := envAccessToken()
	if err != nil {
		return "", err
	}
	if token != "" {
		return token, nil
	}

	settings, err := settings.ReadSettings()
	if err != nil {
		return "", fmt.Errorf("could not read token from settings file: %w", err)
	}

	token = settings.GetToken()
	if !isJwtTokenValid(token) {
		return "", ErrNotLoggedIn
	}

	return token, nil
}

func envAccessToken() (string, error) {
	token := os.Getenv(ENV_ACCESS_TOKEN)
	if token == "" {
		return "", nil
	}
	if !isJwtTokenValid(token) {
		return "", fmt.Errorf("token in %s env var is invalid. Update the env var with a valid value, or unset it to use a token from the configuration file", ENV_ACCESS_TOKEN)
	}
	return token, nil
}

func locations(client *turso.Client) (map[string]string, error) {
	settings, _ := settings.ReadSettings()
	return readLocations(settings, client)
}

func readLocations(settings *settings.Settings, client *turso.Client) (map[string]string, error) {
	if locations := locationsCache(); locations != nil {
		return locations, nil
	}

	locationsMap, err := mapLocations(client)
	if err != nil {
		return nil, err
	}

	locations := make(map[string]string, 32)
	for _, platformLocations := range locationsMap {
		for loc, desc := range platformLocations {
			locations[loc] = desc
		}
	}

	setLocationsCache(locations)
	return locations, nil
}

func mapLocations(client *turso.Client) (map[string]map[string]string, error) {
	locations, err := client.Organizations.Locations()
	if err != nil {
		return nil, err
	}
	return locations, nil
}

func closestLocation(client *turso.Client) (string, error) {
	if closest := closestLocationCache(); closest != "" {
		return closest, nil
	}

	closest, err := client.Locations.Closest()
	if err != nil {
		// We fallback to ams if we are unable to probe the closest location.
		return "ams", err
	}

	setClosestLocationCache(closest)
	return closest, nil
}

func isValidLocation(client *turso.Client, location string) bool {
	locations, err := locations(client)
	if err != nil {
		return true
	}
	_, ok := locations[location]
	return ok
}

func formatLocation(client *turso.Client, id string) string {
	locations, _ := locations(client)
	if desc, ok := locations[id]; ok {
		return fmt.Sprintf("%s (%s)", desc, id)
	}
	return fmt.Sprintf("Location ID: %s", id)
}
