package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/fatih/color"
	"github.com/lucasepe/codename"
	"github.com/spf13/cobra"
)

// Color function for emphasising text.
var emph = color.New(color.FgBlue, color.Bold).SprintFunc()

var region string
var regionIds = []string{
	"ams",
	"cdg",
	"den",
	"dfw",
	"ewr",
	"fra",
	"gru",
	"hkg",
	"iad",
	"jnb",
	"lax",
	"lhr",
	"maa",
	"mad",
	"mia",
	"nrt",
	"ord",
	"otp",
	"scl",
	"sea",
	"sin",
	"sjc",
	"syd",
	"waw",
	"yul",
	"yyz",
}

func extractDatabaseNames(databases []interface{}) []string {
	names := make([]string, 0)
	for _, database := range databases {
		db := database.(map[string]interface{})
		name := db["Name"]
		ty := db["Type"]
		if ty == "primary" {
			names = append(names, name.(string))
		}
	}
	return names
}

func fetchDatabaseNames() []string {
	databases, err := getDatabases()
	if err != nil {
		return []string{}
	}
	return extractDatabaseNames(databases)
}

func getDatabaseNames() []string {
	settings, err := settings.ReadSettings()
	if err != nil {
		return fetchDatabaseNames()
	}
	cached_names := settings.GetDbNamesCache()
	if cached_names != nil {
		return cached_names
	}
	names := fetchDatabaseNames()
	settings.SetDbNamesCache(names)
	return names
}

func getDatabases() ([]interface{}, error) {
	accessToken, err := getAccessToken()
	if err != nil {
		return nil, err
	}
	host := getHost()
	url := fmt.Sprintf("%s/v1/databases", host)
	bearer := "Bearer " + accessToken
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", bearer)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to get database listing: %s", resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.(map[string]interface{})["databases"].([]interface{}), nil
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(createCmd, shellCmd, destroyCmd, replicateCmd, listCmd, regionsCmd)
	createCmd.Flags().StringVar(&region, "region", "", "Region ID. If no ID is specified, closest region to you is used by default.")
	createCmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return regionIds, cobra.ShellCompDirectiveDefault
	})
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
		return "", fmt.Errorf("user not logged in")
	}

	return token, nil
}

func getHost() string {
	host := os.Getenv("IKU_API_HOSTNAME")
	if host == "" {
		host = "https://api.chiseledge.com"
	}
	return host
}

var createCmd = &cobra.Command{
	Use:               "create [flags] [database_name]",
	Short:             "Create a database.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 0 || args[0] == "" {
			rng, err := codename.DefaultRNG()
			if err != nil {
				return err
			}
			name = codename.Generate(rng, 0)
		} else {
			name = args[0]
		}
		region := region
		if region == "" {
			region = probeClosestRegion()
		}
		accessToken, err := getAccessToken()
		if err != nil {
			return err
		}
		host := getHost()
		url := fmt.Sprintf("%s/v1/databases", host)
		bearer := "Bearer " + accessToken
		createDbReq := []byte(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, name, region))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(createDbReq))
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", bearer)
		s := spinner.New(spinner.CharSets[36], 800*time.Millisecond)
		regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
		s.Prefix = fmt.Sprintf("Creating database %s to %s ", emph(name), emph(regionText))
		s.Start()
		start := time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)
		s.Stop()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Failed to create database: %s", resp.Status)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		end := time.Now()
		elapsed := end.Sub(start)
		m := result.(map[string]interface{})["database"].(map[string]interface{})
		username := result.(map[string]interface{})["username"].(string)
		password := result.(map[string]interface{})["password"].(string)
		dbHost := m["Hostname"].(string)
		pgUrl := fmt.Sprintf("postgresql://%v", dbHost)
		fmt.Printf("Created database %s to %s in %d seconds.\n\n", emph(name), emph(regionText), int(elapsed.Seconds()))
		fmt.Printf("You can access the database by running:\n\n")
		fmt.Printf("   psql %s\n\n", pgUrl)
		dbSettings := settings.DatabaseSettings{
			Host:     dbHost,
			Username: username,
			Password: password,
		}
		dbUrl := dbSettings.GetURL()
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("\n")
		config.AddDatabase(name, &dbSettings)
		config.InvalidateDbNamesCache()
		return nil
	},
}

// The fallback region ID to use if we are unable to probe the closest region.
const FallbackRegionId = "ams"

func probeClosestRegion() string {
	probeUrl := "http://api.fly.io"
	resp, err := http.Get(probeUrl)
	if err != nil {
		return FallbackRegionId
	}
	rawRequestId := resp.Header["Fly-Request-Id"]
	if len(rawRequestId) == 0 {
		return FallbackRegionId
	}
	requestId := strings.Split(rawRequestId[0], "-")
	if len(requestId) < 2 {
		return FallbackRegionId
	}
	closestRegionId := requestId[1]
	// Fly has regions that are not available to users. So let's ensure
	// that we return a region ID that is actually usable for provisioning
	// a database.
	for _, regionId := range regionIds {
		if closestRegionId == regionId {
			return closestRegionId
		}
	}
	return FallbackRegionId
}

func destroyArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var destroyCmd = &cobra.Command{
	Use:               "destroy database_name",
	Short:             "Destroy a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: destroyArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("You must specify a database name to delete it.")
		}

		accessToken, err := getAccessToken()
		if err != nil {
			return fmt.Errorf("please login with %s", emph("turso auth login"))
		}
		host := os.Getenv("IKU_API_HOSTNAME")
		if host == "" {
			host = "https://api.chiseledge.com"
		}
		url := fmt.Sprintf("%s/v1/databases/%s", host, name)
		bearer := "Bearer " + accessToken
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", bearer)
		s := spinner.New(spinner.CharSets[14], 40*time.Millisecond)
		s.Prefix = fmt.Sprintf("Destroying database %s... ", emph(name))
		s.Start()
		start := time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)
		s.Stop()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Failed to destroy database: %s", resp.Status)
		}
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Printf("Destroyed database %s in %d seconds.\n", emph(name), int(elapsed.Seconds()))
		settings, err := settings.ReadSettings()
		if err == nil {
			settings.InvalidateDbNamesCache()
		}
		return nil
	},
}

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		return regionIds, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 0 {
		return getDatabaseNames(), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var replicateCmd = &cobra.Command{
	Use:               "replicate database_name region_id",
	Short:             "Replicate a database.",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: replicateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		name := args[0]
		if name == "" {
			return fmt.Errorf("You must specify a database name to replicate it.")
		}
		region := args[1]
		if region == "" {
			return fmt.Errorf("You must specify a database region ID to replicate it.")
		}
		accessToken, err := getAccessToken()
		if err != nil {
			return fmt.Errorf("please login with %s", emph("turso auth login"))
		}
		host := os.Getenv("IKU_API_HOSTNAME")
		if host == "" {
			host = "https://api.chiseledge.com"
		}
		url := fmt.Sprintf("%s/v1/databases", host)
		bearer := "Bearer " + accessToken
		createDbReq := []byte(fmt.Sprintf(`{"name": "%s", "region": "%s", "type": "replica"}`, name, region))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(createDbReq))
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", bearer)
		s := spinner.New(spinner.CharSets[36], 800*time.Millisecond)
		regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
		s.Prefix = fmt.Sprintf("Replicating database %s to %s ", emph(name), emph(regionText))
		s.Start()
		start := time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)
		s.Stop()
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Failed to create database: %s", resp.Status)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var result interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return err
		}
		end := time.Now()
		elapsed := end.Sub(start)
		m := result.(map[string]interface{})["database"].(map[string]interface{})
		username := result.(map[string]interface{})["username"].(string)
		password := result.(map[string]interface{})["password"].(string)
		dbHost := m["Hostname"].(string)
		pgUrl := fmt.Sprintf("postgresql://%v", dbHost)
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", emph(name), emph(regionText), int(elapsed.Seconds()))
		fmt.Printf("You can access the database by running:\n\n")
		fmt.Printf("   psql %s\n", pgUrl)
		dbSettings := settings.DatabaseSettings{
			Host:     dbHost,
			Username: username,
			Password: password,
		}
		dbUrl := dbSettings.GetURL()
		fmt.Printf("\n")
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("\n")
		config.InvalidateDbNamesCache()
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:               "list",
	Short:             "List databases.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		databases, err := getDatabases()
		if err != nil {
			return err
		}
		nameWidth := 8
		regionWidth := 8
		for _, database := range databases {
			db := database.(map[string]interface{})
			name := db["Name"].(string)
			nameLen := len(name)
			if nameWidth < nameLen {
				nameWidth = nameLen
			}
			region := db["Region"].(string)
			regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
			regionLen := len(regionText)
			if regionWidth < regionLen {
				regionWidth = regionLen
			}
		}
		typeWidth := 7
		hostWidth := 15
		fmt.Printf("%-*s  %-*s  %-*s %-*s  %s\n", nameWidth, "NAME", typeWidth, "TYPE", hostWidth, "HOST", regionWidth, "REGION", "URL")
		for _, database := range databases {
			db := database.(map[string]interface{})
			name := db["Name"].(string)
			ty := db["Type"]
			host := db["Hostname"]
			region := db["Region"].(string)
			dbSettings := settings.GetDatabaseSettings(name)
			var url string
			if dbSettings != nil {
				url = dbSettings.GetURL()
			} else {
				url = "<n/a>"
			}
			regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
			fmt.Printf("%-*s  %-*s  %-*s %-*s  %s\n", nameWidth, name, typeWidth, ty, hostWidth, host, regionWidth, regionText, url)
		}
		settings.SetDbNamesCache(extractDatabaseNames(databases))
		return nil
	},
}

var regionsCmd = &cobra.Command{
	Use:               "regions",
	Short:             "List available database regions.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	Run: func(cmd *cobra.Command, args []string) {
		defaultRegionId := probeClosestRegion()
		fmt.Println("ID   LOCATION")
		for _, regionId := range regionIds {
			suffix := ""
			if regionId == defaultRegionId {
				suffix = "  [default]"
			}
			line := fmt.Sprintf("%s  %s%s", regionId, toLocation(regionId), suffix)
			if regionId == defaultRegionId {
				line = emph(line)
			}
			fmt.Printf("%s\n", line)
		}
	},
}

func toLocation(regionId string) string {
	switch regionId {
	case "ams":
		return "Amsterdam, Netherlands"
	case "cdg":
		return "Paris, France"
	case "den":
		return "Denver, Colorado (US)"
	case "dfw":
		return "Dallas, Texas (US)"
	case "ewr":
		return "Secaucus, NJ (US)"
	case "fra":
		return "Frankfurt, Germany"
	case "gru":
		return "São Paulo, Brazil"
	case "hkg":
		return "Hong Kong, Hong Kong"
	case "iad":
		return "Ashburn, Virginia (US)"
	case "jnb":
		return "Johannesburg, South Africa"
	case "lax":
		return "Los Angeles, California (US)"
	case "lhr":
		return "London, United Kingdom"
	case "maa":
		return "Chennai (Madras), India"
	case "mad":
		return "Madrid, Spain"
	case "mia":
		return "Miami, Florida (US)"
	case "nrt":
		return "Tokyo, Japan"
	case "ord":
		return "Chicago, Illinois (US)"
	case "otp":
		return "Bucharest, Romania"
	case "scl":
		return "Santiago, Chile"
	case "sea":
		return "Seattle, Washington (US)"
	case "sin":
		return "Singapore"
	case "sjc":
		return "Sunnyvale, California (US)"
	case "syd":
		return "Sydney, Australia"
	case "waw":
		return "Warsaw, Poland"
	case "yul":
		return "Montreal, Canada"
	case "yyz":
		return "Toronto, Canada"
	default:
		return fmt.Sprintf("Region ID: %s", regionId)
	}
}
