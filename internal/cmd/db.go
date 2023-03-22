package cmd

import (
	"fmt"
	"os"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showUrlFlag bool
var showHttpUrlFlag bool
var showInstanceUrlFlag string
var passwordFlag string
var yesFlag bool
var instanceFlag string

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

func completeInstanceName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		return getInstanceNames(createTursoClient(), args[0]), cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(shellCmd, listCmd, regionsCmd, showCmd, dbInspectCmd, changePasswordCmd, dbAuthCmd)
	showCmd.Flags().BoolVar(&showUrlFlag, "url", false, "Show URL for the database HTTP API.")
	showCmd.Flags().StringVar(&showInstanceUrlFlag, "instance-url", "", "Show URL for the HTTP API of a selected instance of a database. Instance is selected by instance name.")
	showCmd.RegisterFlagCompletionFunc("instance-url", completeInstanceName)
	showCmd.RegisterFlagCompletionFunc("instance-ws-url", completeInstanceName)

	changePasswordCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Value of new password to be set on database")
	changePasswordCmd.RegisterFlagCompletionFunc("password", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

const ENV_ACCESS_TOKEN = "TURSO_API_TOKEN"

func getAccessToken() (string, error) {
	envToken := os.Getenv(ENV_ACCESS_TOKEN)
	if envToken != "" {
		return envToken, nil
	}
	flagToken := viper.GetString("token")
	if flagToken != "" {
		return flagToken, nil
	}
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

// The fallback region ID to use if we are unable to probe the closest location.
const FallbackRegionId = "ams"

const FallbackWarning = "Warning: we could not determine the deployment location closest to your physical location.\nThe location is defaulting to Amsterdam (ams). Consider specifying a location to select a better option using\n\n\tturso db create --location [location].\n\nRun turso db locations for a list of supported locations.\n"

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
	return fmt.Sprintf("Location ID: %s", regionId)
}
