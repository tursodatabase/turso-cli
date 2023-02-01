package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/fatih/color"
	"github.com/lucasepe/codename"
	"github.com/spf13/cobra"
)

// Color function for emphasising text.
var emph = color.New(color.FgBlue, color.Bold).SprintFunc()

var warn = color.New(color.FgYellow, color.Bold).SprintFunc()

var canary bool
var force bool
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

func extractDatabaseNames(databases []turso.Database) []string {
	names := make([]string, 0)
	for _, database := range databases {
		name := database.Name
		ty := database.Type
		if ty == "primary" {
			names = append(names, name)
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

func getDatabase(name string) (turso.Database, error) {
	databases, err := getDatabases()
	if err != nil {
		return turso.Database{}, err
	}

	for _, database := range databases {
		if database.Name == name {
			return database, nil
		}
	}

	return turso.Database{}, fmt.Errorf("database with name %s not found", name)
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

func getDatabases() ([]turso.Database, error) {
	return turso.Databases.List()
}

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(createCmd, shellCmd, destroyCmd, replicateCmd, listCmd, regionsCmd, dropCmd, showCmd)
	createCmd.Flags().BoolVar(&canary, "canary", false, "Use database canary build.")
	createCmd.Flags().StringVar(&region, "region", "", "Region ID. If no ID is specified, closest region to you is used by default.")
	createCmd.RegisterFlagCompletionFunc("region", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return regionIds, cobra.ShellCompDirectiveDefault
	})
	destroyCmd.Flags().BoolVar(&force, "force", false, "Force deletion.")
	replicateCmd.Flags().BoolVar(&canary, "canary", false, "Use database canary build.")
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
	host := os.Getenv("TURSO_API_BASEURL")
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
		var image string
		if canary {
			image = "canary"
		} else {
			image = "latest"
		}
		accessToken, err := getAccessToken()
		if err != nil {
			return err
		}
		host := getHost()
		url := fmt.Sprintf("%s/v2/databases", host)
		bearer := "Bearer " + accessToken
		createDbReq := []byte(fmt.Sprintf(`{"name": "%s", "region": "%s", "image": "%s"}`, name, region, image))
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
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnprocessableEntity {
			return fmt.Errorf("Database name '%s' is not available", emph(name))
		}

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				var result interface{}
				if err := json.Unmarshal(body, &result); err == nil {
					return fmt.Errorf("Failed to create database: %s [%s]", resp.Status, result.(map[string]interface{})["error"])
				}
			}
			return fmt.Errorf("Failed to create database: %s", resp.Status)
		}

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
		dbId := m["DbId"].(string)
		dbHost := m["Hostname"].(string)
		fmt.Printf("Created database %s to %s in %d seconds.\n\n", emph(name), emph(regionText), int(elapsed.Seconds()))
		dbSettings := settings.DatabaseSettings{
			Name:     name,
			Host:     dbHost,
			Username: username,
			Password: password,
		}
		fmt.Printf("HTTP connection string:\n\n")
		dbUrl := dbSettings.GetURL()
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", name)
		config.AddDatabase(dbId, &dbSettings)
		config.InvalidateDbNamesCache()
		return nil
	},
}

// The fallback region ID to use if we are unable to probe the closest region.
const FallbackRegionId = "ams"

const FallbackWarning = "Warning: we could not determine the deployment region closest to your physical location.\nThe region is defaulting to Amsterdam (ams). Consider specifying a region to select a better option using\n\n\tturso db create --region [region].\n\nRun turso db regions for a list of supported regions.\n"

type Region struct {
	Server string
}

func probeClosestRegion() string {
	probeUrl := "https://chisel-region.fly.dev"
	resp, err := http.Get(probeUrl)
	if err != nil {
		fmt.Printf(warn(FallbackWarning))
		return FallbackRegionId
	}
	defer resp.Body.Close()

	reg := Region{}
	err = json.NewDecoder(resp.Body).Decode(&reg)
	if err != nil {
		return FallbackRegionId
	}

	// Fly has regions that are not available to users. So let's ensure
	// that we return a region ID that is actually usable for provisioning
	// a database.
	if !isValidRegion(reg.Server) {
		return reg.Server
	}
	return FallbackRegionId
}

func isValidRegion(region string) bool {
	for _, regionId := range regionIds {
		if region == regionId {
			return true
		}
	}
	return false
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

		start := time.Now()
		s := startSpinner(fmt.Sprintf("Destroying database %s... ", emph(name)))
		if err := turso.Databases.Delete(name); err != nil {
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
	},
}

var dropCmd = &cobra.Command{
	Use:   "drop database_name region",
	Short: "Drop a database from a given region.",
	Args: cobra.MatchAll(
		cobra.ExactArgs(2),
		dbNameValidator(0),
		regionArgValidator(1),
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := getDatabase(args[0])
		if err != nil {
			return err
		}

		region := args[1]
		if db.Type != "logical" {
			return fmt.Errorf("database '%s' does not support the drop operation", db.Name)
		}

		instances, err := turso.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		instance := findInstanceFromRegion(instances, region)
		if instance == nil {
			return fmt.Errorf("could not find any instance of database %s on region %s", db.Name, region)
		}

		err = turso.Instances.Delete(db.Name, instance.Name)
		if err != nil {
			// TODO: remove this once wait stopped bug is fixed
			time.Sleep(3 * time.Second)
			err = turso.Instances.Delete(db.Name, instance.Name)
			if err != nil {
				return fmt.Errorf("could not delete instance %s from region %s: %w", instance.Name, region, err)
			}
		}

		fmt.Printf("Destroyed instance %s in region %s of database %s.\n", emph(instance.Name), emph(region), emph(db.Name))
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show database_name",
	Short: "Show information from a database.",
	Args: cobra.MatchAll(
		cobra.ExactArgs(1),
		dbNameValidator(0),
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := getDatabase(args[0])
		if err != nil {
			return err
		}

		if db.Type != "logical" {
			return fmt.Errorf("database '%s' does not support the show operation", db.Name)
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		instances, err := turso.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		fmt.Println("NAME:   ", db.Name)
		fmt.Println("URL:    ", getDatabaseUrl(config, db))
		fmt.Println("ID:     ", db.ID)
		fmt.Println("REGIONS:", displayRegions(instances))
		fmt.Println()

		data := [][]string{}
		for _, instance := range instances {
			data = append(data, []string{instance.Name, instance.Type, instance.Region})
		}

		fmt.Print("Database Instances:\n")
		printTable("foo", []string{"name", "type", "region"}, data)

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
		if !isValidRegion(region) {
			return fmt.Errorf("Invalid region ID. Run %s to see a list of valid region IDs.", emph("turso db regions"))
		}
		var image string
		if canary {
			image = "canary"
		} else {
			image = "latest"
		}
		accessToken, err := getAccessToken()
		if err != nil {
			return fmt.Errorf("please login with %s", emph("turso auth login"))
		}
		host := getHost()

		original, err := getDatabase(name)
		if err != nil {
			return fmt.Errorf("please login with %s", emph("turso auth login"))
		}

		url := fmt.Sprintf("%s/v1/databases", host)
		if original.Type == "logical" {
			url = fmt.Sprintf("%s/v2/databases/%s/instances", host, name)
		}

		bearer := "Bearer " + accessToken
		createDbReq := []byte(fmt.Sprintf(`{"name": "%s", "region": "%s", "image": "%s", "type": "replica"}`, name, region, image))
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
		var m map[string]interface{}
		if original.Type == "logical" {
			m = result.(map[string]interface{})["instance"].(map[string]interface{})
		} else {
			m = result.(map[string]interface{})["database"].(map[string]interface{})
		}
		username := result.(map[string]interface{})["username"].(string)
		password := result.(map[string]interface{})["password"].(string)
		var dbId, dbHost string
		fmt.Println(original)
		if original.Type == "logical" {
			dbId = m["Uuid"].(string)
			dbHost = original.Hostname
		} else {
			dbId = m["DbId"].(string)
			dbHost = m["Hostname"].(string)
		}
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", emph(name), emph(regionText), int(elapsed.Seconds()))
		dbSettings := settings.DatabaseSettings{
			Host:     dbHost,
			Username: username,
			Password: password,
		}
		fmt.Printf("HTTP connection string:\n\n")
		dbUrl := dbSettings.GetURL()
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", dbUrl)
		config.AddDatabase(dbId, &dbSettings)
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
			name := database.Name
			nameLen := len(name)
			if nameWidth < nameLen {
				nameWidth = nameLen
			}
			region := database.Region
			regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
			regionLen := len(regionText)
			if regionWidth < regionLen {
				regionWidth = regionLen
			}
		}
		typeWidth := 12
		fmt.Printf("%-*s  %-*s %-*s  %s\n", nameWidth, "NAME", typeWidth, "TYPE", regionWidth, "REGION", "URL")
		for _, database := range databases {
			name := database.Name
			ty := database.Type
			region := database.Region
			url := getDatabaseUrl(settings, database)
			regionText := fmt.Sprintf("%s (%s)", toLocation(region), region)
			fmt.Printf("%-*s  %-*s %-*s  %s\n", nameWidth, name, typeWidth, ty, regionWidth, regionText, url)
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
