package cmd

import (
	"fmt"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var canary bool
var showUrlFlag bool
var region string
var yesFlag bool
var instanceFlag string
var regionFlag string

func getRegionIds(client *turso.Client) []string {
	settings, err := settings.ReadSettings()
	var cached_names []string
	if err == nil {
		cached_names = settings.GetRegionsCache()
		if cached_names != nil {
			return cached_names
		}
	}
	regions, err := turso.GetRegions(client)
	if err != nil {
		return []string{}
	}
	if settings != nil {
		settings.SetRegionsCache(regions.Ids)
	}
	return regions.Ids
}

func extractDatabaseNames(databases []turso.Database) []string {
	names := make([]string, 0)
	for _, database := range databases {
		name := database.Name
		ty := database.Type
		if ty == "logical" {
			names = append(names, name)
		}
	}
	return names
}

func fetchDatabaseNames(client *turso.Client) []string {
	databases, err := getDatabases(client)
	if err != nil {
		return []string{}
	}
	return extractDatabaseNames(databases)
}

func getDatabase(client *turso.Client, name string) (turso.Database, error) {
	databases, err := getDatabases(client)
	if err != nil {
		return turso.Database{}, err
	}

	for _, database := range databases {
		if database.Name == name {
			return database, nil
		}
	}

	return turso.Database{}, fmt.Errorf("database %s not found. List known databases using %s", turso.Emph(name), turso.Emph("turso db list"))
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

func getDatabases(client *turso.Client) ([]turso.Database, error) {
	return client.Databases.List()
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(createCmd, shellCmd, destroyCmd, replicateCmd, listCmd, regionsCmd, showCmd)
	destroyCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the destruction of all regions of the database.")
	destroyCmd.Flags().StringVar(&regionFlag, "region", "", "Pick a database region to destroy.")
	destroyCmd.Flags().StringVar(&instanceFlag, "instance", "", "Pick a specific database instance to destroy.")
	createCmd.Flags().BoolVar(&canary, "canary", false, "Use database canary build.")
	createCmd.Flags().StringVar(&region, "region", "", "Region ID. If no ID is specified, closest region to you is used by default.")
	createCmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getRegionIds(createTursoClient()), cobra.ShellCompDirectiveDefault
	})
	replicateCmd.Flags().BoolVar(&canary, "canary", false, "Use database canary build.")
	showCmd.Flags().BoolVar(&showUrlFlag, "url", false, "Show database connection URL.")
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

func getAccessToken() (string, error) {
	settings, err := settings.ReadSettings()
	if err != nil {
		return "", fmt.Errorf("could not read local settings")
	}

	token := settings.GetToken()
	if token == "" {
		return "", fmt.Errorf("user not logged in, please use %s", turso.Emph("turso auth login"))
	}

	return token, nil
}

func getHost() string {
	host := os.Getenv("TURSO_API_BASEURL")
	if host == "" {
		host = "https://api.chiseledge.com"
	}
	return host
}

// The fallback region ID to use if we are unable to probe the closest region.
const FallbackRegionId = "ams"

const FallbackWarning = "Warning: we could not determine the deployment region closest to your physical location.\nThe region is defaulting to Amsterdam (ams). Consider specifying a region to select a better option using\n\n\tturso db create --region [region].\n\nRun turso db regions for a list of supported regions.\n"

type Region struct {
	Server string
}

func probeClosestRegion() string {
	region := turso.GetDefaultRegion()
	if region == "" {
		return FallbackRegionId
	}
	return region
}

func isValidRegion(client *turso.Client, region string) bool {
	regionIds := getRegionIds(client)
	if len(regionIds) == 0 {
		return true
	}
	for _, regionId := range regionIds {
		if region == regionId {
			return true
		}
	}
	return false
}

func toLocation(client *turso.Client, regionId string) string {
	regions, err := turso.GetRegions(client)
	if err == nil {
		for idx := range regions.Ids {
			if regions.Ids[idx] == regionId {
				return regions.Descriptions[idx]
			}
		}
	}
	return fmt.Sprintf("Region ID: %s", regionId)
}
