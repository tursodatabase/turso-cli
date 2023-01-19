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
	"github.com/fatih/color"
	"github.com/lucasepe/codename"
	"github.com/spf13/cobra"
)

var Region string

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(createCmd, destroyCmd, replicateCmd, listCmd, regionsCmd)
	createCmd.Flags().StringVar(&Region, "region", "", "Region ID. If no ID is specified, closest region to you is used by default.")
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

func getAccessToken() (string, error) {
	accessToken := os.Getenv("IKU_API_TOKEN")
	if accessToken == "" {
		return "", fmt.Errorf("please set the `IKU_API_TOKEN` environment variable to your access token")
	}
	return accessToken, nil
}

func getHost() string {
	host := os.Getenv("IKU_API_HOSTNAME")
	if host == "" {
		host = "https://api.chiseledge.com"
	}
	return host
}

var createCmd = &cobra.Command{
	Use:   "create [flags] [database_name]",
	Short: "Create a database.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		region := Region
		if region == "" {
			region = "ams"
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
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Creating database `%s` to %s... ", name, toLocation(region))
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
		dbHost := m["Host"].(string)
		dbRegion := m["Region"].(string)
		pgUrl := fmt.Sprintf("postgresql://%v", dbHost)
		fmt.Printf("Created database `%s` to %s in %d seconds.\n\n", name, toLocation(dbRegion), int(elapsed.Seconds()))
		fmt.Printf("You can access the database by running:\n\n")
		fmt.Printf("   psql %s\n\n", pgUrl)
		fmt.Printf("or via HTTP API:\n\n")
		fmt.Printf("   http://%s:%s@%s\n\n", username, password, dbHost)
		fmt.Printf("\n")
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
	if rawRequestId == nil || len(rawRequestId) == 0 {
		return FallbackRegionId
	}
	requestId := strings.Split(rawRequestId[0], "-")
	if len(requestId) < 2 {
		return FallbackRegionId
	}
	return requestId[1]
}

var destroyCmd = &cobra.Command{
	Use:   "destroy database_name",
	Short: "Destroy a database.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("You must specify a database name to delete it.")
		}
		accessToken := os.Getenv("IKU_API_TOKEN")
		if accessToken == "" {
			return fmt.Errorf("please set the `IKU_API_TOKEN` environment variable to your access token")
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
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Destroying database `%s`... ", name)
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
		fmt.Printf("Destroyed database `%s` in %d seconds.\n", name, int(elapsed.Seconds()))
		return nil
	},
}

var replicateCmd = &cobra.Command{
	Use:   "replicate database_name region_id",
	Short: "Replicate a database.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("You must specify a database name to replicate it.")
		}
		region := args[1]
		if region == "" {
			return fmt.Errorf("You must specify a database region ID to replicate it.")
		}
		accessToken := os.Getenv("IKU_API_TOKEN")
		if accessToken == "" {
			return fmt.Errorf("please set the `IKU_API_TOKEN` environment variable to your access token")
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
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Prefix = fmt.Sprintf("Replicating database `%s` to %s... ", name, toLocation(region))
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
		dbHost := m["Host"].(string)
		dbRegion := m["Region"].(string)
		pgUrl := fmt.Sprintf("postgresql://%v", dbHost)
		fmt.Printf("Replicated database `%s` to %s in %d seconds.\n\n", name, toLocation(dbRegion), int(elapsed.Seconds()))
		fmt.Printf("You can access the database by running:\n\n")
		fmt.Printf("   psql %s\n", pgUrl)
		fmt.Printf("\n")
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List databases.",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := getAccessToken()
		if err != nil {
			return err
		}
		host := getHost()
		url := fmt.Sprintf("%s/v1/databases", host)
		bearer := "Bearer " + accessToken
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", bearer)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Failed to get database listing: %s", resp.Status)
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
		databases := result.(map[string]interface{})["databases"].([]interface{})
		nameWidth := 8
		for _, database := range databases {
			db := database.(map[string]interface{})
			name := db["Name"].(string)
			nameLen := len(name)
			if nameWidth < nameLen {
				nameWidth = nameLen
			}
		}
		typeWidth := 7
		hostWidth := 15
		fmt.Printf("%-*s  %-*s  %-*s\n", nameWidth, "NAME", typeWidth, "TYPE", hostWidth, "HOST")
		for _, database := range databases {
			db := database.(map[string]interface{})
			name := db["Name"]
			ty := db["Type"]
			host := db["Host"]
			fmt.Printf("%-*s  %s  %s\n", nameWidth, name, ty, host)
		}
		return nil
	},
}

var regionsCmd = &cobra.Command{
	Use:   "regions",
	Short: "List available database regions.",
	Run: func(cmd *cobra.Command, args []string) {
		defaultRegionId := probeClosestRegion()
		regionIds := []string{
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
		fmt.Println("ID   LOCATION")
		for _, regionId := range regionIds {
			suffix := ""
			if regionId == defaultRegionId {
				suffix = "  [default]"
			}
			line := fmt.Sprintf("%s  %s%s", regionId, toLocation(regionId), suffix)
			if regionId == defaultRegionId {
				emph := color.New(color.FgWhite, color.Bold).SprintFunc()
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
