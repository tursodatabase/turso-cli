package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/briandowns/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 1 {
		return getRegionIds(createTursoClient()), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var replicateCmd = &cobra.Command{
	Use:               "replicate database_name region_id",
	Short:             "Replicate a database.",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: replicateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("you must specify a database name to replicate it")
		}
		region := args[1]
		if region == "" {
			return fmt.Errorf("you must specify a database region ID to replicate it")
		}
		cmd.SilenceUsage = true
		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		tursoClient := createTursoClient()
		if !isValidRegion(tursoClient, region) {
			return fmt.Errorf("invalid region ID. Run %s to see a list of valid region IDs", turso.Emph("turso db regions"))
		}
		var image string
		if canary {
			image = "canary"
		} else {
			image = "latest"
		}
		accessToken, err := getAccessToken()
		if err != nil {
			return fmt.Errorf("please login with %s", turso.Emph("turso auth login"))
		}
		host := getHost()

		original, err := getDatabase(tursoClient, name)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("%s/v1/databases", host)
		if original.Type == "logical" {
			url = fmt.Sprintf("%s/v2/databases/%s/instances", host, name)
		}

		bearer := "Bearer " + accessToken
		dbSettings := config.GetDatabaseSettings(original.ID)
		password := dbSettings.Password

		createDbReq := []byte(fmt.Sprintf(`{"name": "%s", "region": "%s", "image": "%s", "type": "replica", "password": "%s"}`, name, region, image, password))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(createDbReq))
		if err != nil {
			return err
		}
		req.Header.Add("Authorization", bearer)
		s := spinner.New(spinner.CharSets[36], 800*time.Millisecond)
		regionText := fmt.Sprintf("%s (%s)", toLocation(tursoClient, region), region)
		s.Prefix = fmt.Sprintf("Replicating database %s to %s ", turso.Emph(name), turso.Emph(regionText))
		s.Start()
		start := time.Now()
		client := &http.Client{}
		resp, err := client.Do(req)
		s.Stop()
		if err != nil {
			return fmt.Errorf("failed to create database: %s", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to create database: %s", resp.Status)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
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
		password = result.(map[string]interface{})["password"].(string)
		var dbHost string
		if original.Type == "logical" {
			dbHost = original.Hostname
		} else {
			dbHost = m["Hostname"].(string)
		}
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", turso.Emph(name), turso.Emph(regionText), int(elapsed.Seconds()))
		dbSettings = &settings.DatabaseSettings{
			Host:     dbHost,
			Username: username,
			Password: password,
		}
		fmt.Printf("HTTP connection string:\n\n")
		dbUrl := dbSettings.GetURL()
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", dbUrl)
		return nil
	},
}
