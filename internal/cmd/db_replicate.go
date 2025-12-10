package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/exp/maps"
)

func init() {
	dbCmd.AddCommand(replicateCmd)
	addWaitFlag(replicateCmd, "Wait for the replica to be ready to receive requests.")
}

var replicateCmd = &cobra.Command{
	Use:               "replicate <database-name> <location-code>",
	Short:             "Replicate a database.",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: replicateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		dbName := args[0]
		if dbName == "" {
			return errors.New("you must specify a database name to replicate it")
		}

		database, err := getDatabase(client, dbName, true)
		if err != nil {
			return err
		}
		if strings.HasPrefix(database.PrimaryRegion, "aws-") {
			return errors.New("replication is not available on AWS at the moment")
		}

		location, err := getReplicateLocation(client, args, database)
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		if !isValidLocation(client, location) {
			return fmt.Errorf("invalid location ID. Run %s to see a list of valid location IDs", internal.Emph("turso db locations"))
		}

		if ok, _ := canReplicate(client, dbName); !ok {
			cmd := internal.Emph(fmt.Sprintf("turso group locations add %s %s", database.Group, location))
			return fmt.Errorf("database %s is part of a group.\nUse %s to replicate the group instead", internal.Emph(dbName), cmd)
		}

		instance, err := replicate(client, database, location)
		if err != nil {
			return err
		}

		showCmd := fmt.Sprintf("turso db show %s", dbName)
		urlCmd := fmt.Sprintf("turso db show %s --instance-url %s", dbName, instance.Name)
		fmt.Printf("To see information about the database %s, run:\n\n\t%s\n\n", internal.Emph(dbName), internal.Emph(showCmd))
		fmt.Printf("To see a connection URL directly to the new replica, run:\n\n\t%s\n\n", internal.Emph(urlCmd))

		invalidateDatabasesCache()
		return nil
	},
}

func replicate(client *turso.Client, database turso.Database, location string) (*turso.Instance, error) {
	start := time.Now()
	instance, err := createInstance(client, database, location)
	if shouldRetryReplicate(err) {
		location, err = selectAlternativeLocation(client, database.Name, location)
		if err != nil {
			return nil, err
		}
		start = time.Now()
		instance, err = createInstance(client, database, location)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to replicate database: %s", err)
	}

	if waitFlag {
		err := waitForInstance(client, database.Name, instance.Name, location)
		if err != nil {
			return nil, err
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", internal.Emph(database.Name), internal.Emph(formatLocation(client, location)), int(elapsed.Seconds()))
	return instance, nil
}

func waitForInstance(client *turso.Client, database, instance, location string) error {
	description := fmt.Sprintf("Waiting for replica of %s at %s to be ready", internal.Emph(database), internal.Emph(formatLocation(client, location)))
	s := prompt.Spinner(description)
	defer s.Stop()
	return client.Instances.Wait(database, instance)
}

func shouldRetryReplicate(err error) bool {
	var createInstanceLocationError *turso.CreateInstanceLocationError
	return errors.As(err, &createInstanceLocationError)
}

func selectAlternativeLocation(client *turso.Client, database, locationID string) (string, error) {
	fmt.Printf("We couldn't replicate your database to location %s.\nPlease try again in a few moments, or pick one of the nearby locations.\n", internal.Emph(locationID))

	location, _ := client.Locations.Get(locationID)

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

	_, locationID, err := promptSelect.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed %v", err)
	}

	return locationID, nil
}

func createInstance(client *turso.Client, database turso.Database, location string) (*turso.Instance, error) {
	description := fmt.Sprintf("Replicating database %s to %s", internal.Emph(database.Name), internal.Emph(formatLocation(client, location)))
	s := prompt.Spinner(description)
	defer s.Stop()

	if database.Group != "" {
		return &turso.Instance{Name: location, Region: location}, client.Groups.AddLocation(database.Group, location)
	}

	return client.Instances.Create(database.Name, location)
}

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 1 {
		locations, _ := locations(client)
		return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	return dbNameArg(cmd, args, toComplete)
}

func getReplicateLocation(client *turso.Client, args []string, database turso.Database) (string, error) {
	if len(args) > 1 {
		return args[1], nil
	}

	locations, err := locations(client)
	if err != nil {
		return "", err
	}

	location := pickLocation(locations, database.Regions)
	if location == "" {
		return "", errors.New("you must specify a database location ID to replicate it")
	}

	return location, nil
}

func pickLocation(locations map[string]string, exclude []string) string {
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
		if excluded[id] || strings.HasPrefix(id, "aws-") {
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

func canReplicate(client *turso.Client, name string) (bool, error) {
	databases, err := getDatabases(client)
	if err != nil {
		return false, err
	}

	counter := map[string]int{}
	group := ""
	for _, database := range databases {
		counter[database.Group]++
		if database.Name == name {
			group = database.Group
		}
	}

	if group == "" {
		return true, nil
	}
	return counter[group] == 1, nil
}
