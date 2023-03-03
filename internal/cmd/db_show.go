package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
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
			fmt.Println(getDatabaseHttpUrl(config, &db))
			return nil
		}

		if showHranaUrlFlag {
			fmt.Println(getDatabaseHranaUrl(config, &db))
			return nil
		}

		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		if showInstanceUrlFlag != "" || showInstanceHranaUrlFlag != "" {
			for _, instance := range instances {
				if instance.Name == showInstanceUrlFlag {
					fmt.Println(getInstanceHttpUrl(config, &db, &instance))
					return nil
				} else if instance.Name == showInstanceHranaUrlFlag {
					fmt.Println(getInstanceHranaUrl(config, &db, &instance))
					return nil
				}
			}
			return fmt.Errorf("instance %s was not found for database %s. List known instances using %s", turso.Emph(showInstanceUrlFlag), turso.Emph(db.Name), turso.Emph("turso db show "+db.Name))
		}

		regions := make([]string, len(db.Regions))
		copy(regions, db.Regions)
		sort.Strings(regions)

		fmt.Println("Name:      ", db.Name)
		fmt.Println("HTTP URL:  ", getDatabaseHttpUrl(config, &db))
		fmt.Println("Hrana URL: ", getDatabaseHranaUrl(config, &db))
		fmt.Println("ID:        ", db.ID)
		fmt.Println("Regions:   ", strings.Join(regions, ", "))
		fmt.Println()

		versions := [](chan string){}
		httpUrls := []string{}
		for idx, instance := range instances {
			httpUrls = append(httpUrls, getInstanceHttpUrl(config, &db, &instance))
			versions = append(versions, make(chan string, 1))
			go func(idx int) {
				versions[idx] <- fetchInstanceVersion(httpUrls[idx])
			}(idx)
		}

		data := [][]string{}
		for idx, instance := range instances {
			version := <-versions[idx]
			hranaUrl := getInstanceHranaUrl(config, &db, &instance)
			data = append(data, []string{instance.Name, instance.Type, instance.Region, version, httpUrls[idx], hranaUrl})
		}

		fmt.Print("Database Instances:\n")
		printTable([]string{"Name", "Type", "Region", "Version", "HTTP URL", "Hrana URL"}, data)

		return nil
	},
}

func fetchInstanceVersion(baseUrl string) string {
	url, err := url.Parse(baseUrl)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	url.Path = path.Join(url.Path, "/version")
	req, err := http.NewRequest("GET", url.String(), nil)
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
