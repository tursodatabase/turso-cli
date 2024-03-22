package cmd

import (
	"fmt"
	"math"
	"os"
	"sync"

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
	databases, err := client.Databases.List()
	if err != nil {
		return nil, err
	}
	setDatabasesCache(databases)
	return databases, nil
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
		return "", fmt.Errorf("user not logged in, please login with %s", internal.Emph("turso auth login"))
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

type latMap struct {
	id  string
	lat int
}

func locations(client *turso.Client) (map[string]string, error) {
	settings, _ := settings.ReadSettings()
	return readLocations(settings, client)
}

func latencies(client *turso.Client) (map[string]int, error) {
	settings, _ := settings.ReadSettings()
	locations, err := readLocations(settings, client)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	latencies := make(map[string]int)
	c := make(chan latMap, len(locations))
	for id := range locations {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			measure := math.MaxInt
			// XXX: Running this in different goroutines makes all latencies dogslow.
			// Not sure if this is contention at the client or API level
			for i := 0; i < 3; i++ {
				d := turso.ProbeLocation(id)
				if d != nil {
					measure = int(math.Min(float64(d.Milliseconds()), float64(measure)))
				}
			}
			c <- latMap{id: id, lat: measure}
		}(id)
	}

	wg.Wait()
	close(c)

	for kvp := range c {
		latencies[kvp.id] = kvp.lat
	}
	return latencies, nil
}

func readLocations(settings *settings.Settings, client *turso.Client) (map[string]string, error) {
	if locations := locationsCache(); locations != nil {
		return locations, nil
	}

	locations, err := client.Locations.List()
	if err != nil {
		return nil, err
	}

	setLocationsCache(locations)
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
