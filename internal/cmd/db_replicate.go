package cmd

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/manifoldco/promptui"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

func init() {
	dbCmd.AddCommand(replicateCmd)
	addCanaryFlag(replicateCmd)
	addWaitFlag(replicateCmd, "Wait for the replica to be ready to receive requests.")
}

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClientFromAccessToken(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 1 {
		locations, _ := locations(client)
		return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	return dbNameArg(cmd, args, toComplete)
}

func pickLocation(dbName string, locations map[string]string, exclude []string) string {
	fmt.Printf("Pro-tip! Next time, you can pass the location to the command. Invoke it as %s.\n", internal.Emph(fmt.Sprintf("turso db replicate %s [location]", dbName)))
	fmt.Printf("But since we're here... where would you like the replica to be?\n")
	fmt.Printf("%s", internal.Emph("Available locations:\n"))

	excluded := make(map[string]bool)
	for _, key := range exclude {
		excluded[key] = true
	}

	ids := maps.Keys(locations)
	sort.Strings(ids)

	columns := make([]interface{}, 0)
	columns = append(columns, "ID↓")
	columns = append(columns, "LOCATION")

	tbl := table.New(columns...)

	for _, id := range ids {
		if excluded[id] {
			continue
		}
		tbl.AddRow(id, locations[id])
	}
	tbl.Print()
	fmt.Printf("\n%s ", internal.Emph("Your choice:"))
	var choice string
	fmt.Scanf("%s", &choice)
	return choice
}

var replicateCmd = &cobra.Command{
	Use:               "replicate database_name location_id",
	Short:             "Replicate a database.",
	Args:              cobra.RangeArgs(1, 3),
	ValidArgsFunction: replicateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		dbName := args[0]
		if dbName == "" {
			return fmt.Errorf("you must specify a database name to replicate it")
		}

		database, err := getDatabase(client, dbName, true)
		if err != nil {
			return err
		}

		var locationId string
		if len(args) > 1 {
			locationId = args[1]
		} else {
			locations, err := locations(client)
			if err != nil {
				return err
			}
			locationId = pickLocation(dbName, locations, database.Regions)
		}
		if locationId == "" {
			return fmt.Errorf("you must specify a database location ID to replicate it")
		}
		cmd.SilenceUsage = true

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		if !isValidLocation(client, locationId) {
			return fmt.Errorf("invalid location ID. Run %s to see a list of valid location IDs", internal.Emph("turso db locations"))
		}

		image := "latest"
		if canaryFlag {
			image = "canary"
		}

		instanceName := ""
		if len(args) > 2 {
			instanceName = args[2]
		}

		locationText := fmt.Sprintf("%s (%s)", locationDescription(client, locationId), locationId)
		description := fmt.Sprintf("Replicating database %s to %s ", internal.Emph(dbName), internal.Emph(locationText))
		s := prompt.Spinner(description)
		defer s.Stop()

		start := time.Now()
		instance, err := client.Instances.Create(dbName, instanceName, locationId, image)
		var createInstanceLocationError *turso.CreateInstanceLocationError
		if errors.As(err, &createInstanceLocationError) {
			s.Stop()
			instance, description, err = handleDatabaseReplicationError(client, dbName, locationId, image)
			if err != nil {
				return err
			}
			s = prompt.Spinner(description)
			defer s.Stop()
		}
		if err != nil {
			return fmt.Errorf("failed to replicate database: %s", err)
		}

		if waitFlag {
			description := fmt.Sprintf("Waiting for replica of %s in %s to be ready", internal.Emph(dbName), internal.Emph(locationText))
			s.Text(description)
			if err = client.Instances.Wait(dbName, instance.Name); err != nil {
				return err
			}
		}

		s.Stop()
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", internal.Emph(dbName), internal.Emph(locationText), int(elapsed.Seconds()))

		showCmd := fmt.Sprintf("turso db show %s", dbName)
		urlCmd := fmt.Sprintf("turso db show %s --instance-url %s", dbName, instance.Name)
		fmt.Printf("To see information about the database %s, run:\n\n\t%s\n\n", internal.Emph(dbName), internal.Emph(showCmd))
		fmt.Printf("To see a connection URL directly to the new replica, run:\n\n\t%s\n\n", internal.Emph(urlCmd))

		firstTime := config.RegisterUse("db_replicate")
		if firstTime {
			fmt.Println("How is your experience going? We'd love to know!")
			fmt.Printf("🗓   Book a call with us! You can do it with:\n\n\t%s\n", internal.Emph("turso account bookmeeting"))
			fmt.Printf("🎤   Or just send us your feedback:\n\n\t%s\n", internal.Emph("turso account feedback"))
		}
		invalidateDatabasesCache()
		return nil
	},
}

func handleDatabaseReplicationError(client *turso.Client, name, locationId string, image string) (*turso.Instance, string, error) {
	fmt.Printf("We couldn't replicate your database to location %s.\nPlease try again in a few moments, or pick one of the nearby locations we've selected for you.\n", internal.Emph(locationId))

	location, _ := client.Locations.Get(locationId)

	closestLocationCodes := make([]string, 0, len(location.Closest))
	for _, location := range location.Closest {
		code := location.Code
		closestLocationCodes = append(closestLocationCodes, code)
	}
	promptSelect := promptui.Select{
		HideHelp:     true,
		Label:        "Select a location",
		Items:        closestLocationCodes,
		HideSelected: true,
	}

	_, locationId, err := promptSelect.Run()
	if err != nil {
		return nil, "", fmt.Errorf("prompt failed %v", err)
	}

	var instance *turso.Instance
	if instance, err = client.Instances.Create(name, "", locationId, image); err != nil {
		return nil, "", fmt.Errorf("we couldn't replicate your database. Please try again later")
	}

	locationText := fmt.Sprintf("%s (%s)", locationDescription(client, locationId), locationId)
	description := fmt.Sprintf("Replicating database %s to %s ", internal.Emph(name), internal.Emph(locationText))
	return instance, description, nil
}
