package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

func showShellArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var showCmd = &cobra.Command{
	Use:               "show database_name",
	Short:             "Show information from a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: showShellArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		if db.Type != "logical" {
			return fmt.Errorf("only new databases, of type 'logical', support the show operation")
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		if showUrlFlag {
			fmt.Println(getDatabaseUrl(config, db))
			return nil
		}

		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		regions := make([]string, len(db.Regions))
		copy(regions, db.Regions)
		sort.Strings(regions)

		fmt.Println("Name:    ", db.Name)
		fmt.Println("URL:     ", getDatabaseUrl(config, db))
		fmt.Println("ID:      ", db.ID)
		fmt.Println("Regions: ", strings.Join(regions, ", "))
		fmt.Println()

		data := [][]string{}
		for _, instance := range instances {
			url := getInstanceUrl(config, db, instance)
			version := fetchInstanceVersion(url)
			data = append(data, []string{instance.Name, instance.Type, instance.Region, version, url})
		}

		fmt.Print("Database Instances:\n")
		printTable([]string{"name", "type", "region", "version", "url"}, data)

		return nil
	},
}

func fetchInstanceVersion(baseUrl string) string {
	u, err := url.JoinPath(baseUrl, "/version")
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return "0.3.1-"
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	return string(respBody)
}
