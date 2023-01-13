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
	"github.com/lucasepe/codename"
)

type DbCmd struct {
	Create    CreateCmd    `cmd:"" help:"Create a database."`
	Replicate ReplicateCmd `cmd:"" help:"Replicate a database."`
	List      ListCmd      `cmd:"" help:"List databases."`
	Regions   RegionsCmd   `cmd:"" help:"List available database regions."`
}

type CreateCmd struct {
	Name   string `arg:"" optional:"" name:"database name" help:"Database name. If no name is specified, one will be automatically generated."`
	Region string `optional:"" help:"Region ID. If no ID is specified, 'ams' is used by default."`
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

func (cmd *CreateCmd) Run(globals *Globals) error {
	name := cmd.Name
	if name == "" {
		rng, err := codename.DefaultRNG()
		if err != nil {
			return err
		}
		name = codename.Generate(rng, 0)
	}
	region := cmd.Region
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
	dbHost := m["Host"].(string)
	dbRegion := m["Region"].(string)
	pgUrl := fmt.Sprintf("postgresql://%v:5000", dbHost)
	fmt.Printf("Created database `%s` to %s in %d seconds.\n\n", name, toLocation(dbRegion), int(elapsed.Seconds()))
	fmt.Printf("You can access the database by running:\n\n")
	fmt.Printf("   psql %s\n", pgUrl)
	fmt.Printf("\n")
	return nil
}

type ReplicateCmd struct {
	Name   string `arg:"" name:"database name" help:"Database name (required)"`
	Region string `arg:"" name:"region ID" help:"Region ID (required)"`
}

func (cmd *ReplicateCmd) Run(globals *Globals) error {
	name := cmd.Name
	if name == "" {
		return fmt.Errorf("You must specify a database name to replicate it.")
	}
	region := cmd.Region
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
	pgUrl := fmt.Sprintf("postgresql://%v:5000", dbHost)
	fmt.Printf("Replicated database `%s` to %s in %d seconds.\n\n", name, toLocation(dbRegion), int(elapsed.Seconds()))
	fmt.Printf("You can access the database by running:\n\n")
	fmt.Printf("   psql %s\n", pgUrl)
	fmt.Printf("\n")
	return nil
}

type ListCmd struct {
}

func (cmd *ListCmd) Run(globals *Globals) error {
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
}

type RegionsCmd struct {
}

func (cmd *RegionsCmd) Run(globals *Globals) error {
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
	for _, regionId := range regionIds {
		fmt.Printf("  %s - %s\n", regionId, toLocation(regionId))
	}
	return nil
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
		return "São Paulo"
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
