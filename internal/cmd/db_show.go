package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"golang.org/x/sync/errgroup"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showUrlFlag, "url", false, "Show URL for the database HTTP API.")
	showCmd.Flags().BoolVar(&showBasicAuthFlag, "basic-auth", false, "Show basic authentication in the URL.")
	showCmd.Flags().BoolVar(&showInstanceUrlsFlag, "instance-urls", false, "Show URL for the HTTP API of all existing instances")
	showCmd.Flags().StringVar(&showInstanceUrlFlag, "instance-url", "", "Show URL for the HTTP API of a selected instance of a database. Instance is selected by instance name.")
	showCmd.RegisterFlagCompletionFunc("instance-url", completeInstanceName)
	showCmd.RegisterFlagCompletionFunc("instance-ws-url", completeInstanceName)
}

var showCmd = &cobra.Command{
	Use:               "show database_name",
	Short:             "Show information from a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		token, err := client.Databases.Token(db.Name, "1d", true)
		if err != nil {
			return err
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		if showUrlFlag {
			fmt.Println(getDatabaseUrl(config, &db, showBasicAuthFlag))
			return nil
		}

		if showHttpUrlFlag {
			fmt.Println(getDatabaseHttpUrl(config, &db))
			return nil
		}

		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		if showInstanceUrlFlag != "" {
			for _, instance := range instances {
				if instance.Name == showInstanceUrlFlag {
					fmt.Println(getInstanceUrl(config, &db, &instance))
					return nil
				}
			}
			return fmt.Errorf("instance %s was not found for database %s. List known instances using %s", internal.Emph(showInstanceUrlFlag), internal.Emph(db.Name), internal.Emph("turso db show "+db.Name))
		}

		regions := make([]string, len(db.Regions))
		copy(regions, db.Regions)
		sort.Strings(regions)

		versions := [](chan string){}
		urls := []string{}
		inspectRet := InspectInfo{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		results := make(chan *InspectInfo, len(instances))
		for idx, instance := range instances {
			urls = append(urls, getInstanceUrl(config, &db, &instance))
			versions = append(versions, make(chan string, 1))
			go func(idx int, client *turso.Client, config *settings.Settings, db *turso.Database, instance *turso.Instance) {
				versions[idx] <- fetchInstanceVersion(client, config, db, instance)
			}(idx, client, config, &db, &instance)
			loopInstance := instance
			g.Go(func() error {
				url := getInstanceHttpUrl(config, &db, &loopInstance)
				ret, err := inspect(ctx, url, token, loopInstance.Region, verboseFlag)
				if err != nil {
					return err
				}
				results <- ret
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout while inspecting database. It's possible that this database is too old and does not support inspecting or one of the instances is not reachable")
			}
			return err
		}
		for range instances {
			ret := <-results
			inspectRet.Accumulate(ret)
		}
		data := [][]string{}
		for idx, instance := range instances {
			version := <-versions[idx]
			if showInstanceUrlsFlag {
				data = append(data, []string{instance.Name, instance.Type, instance.Region, version, urls[idx]})
			} else {
				data = append(data, []string{instance.Name, instance.Type, instance.Region, version})
			}
		}

		fmt.Println("Name:          ", db.Name)
		fmt.Println("URL:           ", getDatabaseUrl(config, &db, false))
		fmt.Println("ID:            ", db.ID)
		fmt.Println("Locations:     ", strings.Join(regions, ", "))
		fmt.Println("Size:          ", inspectRet.PrintTotal())
		fmt.Println()
		fmt.Print("Database Instances:\n")
		if showInstanceUrlsFlag {
			printTable([]string{"Name", "Type", "Location", "Version", "URL"}, data)
		} else {
			printTable([]string{"Name", "Type", "Location", "Version"}, data)
		}

		return nil
	},
}

func fetchInstanceVersion(client *turso.Client, config *settings.Settings, db *turso.Database, instance *turso.Instance) string {
	baseUrl := getInstanceHttpUrlWithoutAuth(config, db, instance)

	token, err := tokenFromDb(db, client)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}

	if token == "" {
		baseUrl = getInstanceHttpUrl(config, db, instance)
	}
	url, err := url.Parse(baseUrl)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	url.Path = path.Join(url.Path, "/version")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return fmt.Sprintf("fetch failed: %s", err)
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
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
