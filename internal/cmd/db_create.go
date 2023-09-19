package cmd

import (
	"fmt"
	"time"

	"github.com/athoscouto/codename"
	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(createCmd)
	addGroupFlag(createCmd)
	addFromDBFlag(createCmd)
	addDbFromFileFlag(createCmd)
	addLocationFlag(createCmd, "Location ID. If no ID is specified, closest location to you is used by default.")
}

var createCmd = &cobra.Command{
	Use:               "create [flags] [database_name]",
	Short:             "Create a database.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name, err := getDatabaseName(args)
		if err != nil {
			return err
		}

		if err := turso.CheckName(name); err != nil {
			return fmt.Errorf("invalid database name: %w", err)
		}

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		group, err := groupFromFlag(client)
		if err != nil {
			return err
		}

		location, err := locationFromFlag(client)
		if err != nil {
			return err
		}

		timestamp, err := parseTimestampFlag()
		if err != nil {
			return err
		}

		if ok, err := shouldCreateGroup(client, group, location); ok {
			createGroup(client, group, location)
		} else if err != nil {
			return err
		}

		start := time.Now()
		spinner := prompt.Spinner(fmt.Sprintf("Creating database %s in group %s...", internal.Emph(name), internal.Emph(group)))
		defer spinner.Stop()
		if _, err = client.Databases.Create(name, location, "", "", group, fromDBFlag, timestamp); err != nil {
			return fmt.Errorf("could not create database %s: %w", name, err)
		}

		spinner.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Created database %s at group %s in %d seconds.\n\n", internal.Emph(name), internal.Emph(group), int(elapsed.Seconds()))

		fmt.Printf("Start an interactive SQL shell with:\n\n")
		fmt.Printf("   %s\n\n", internal.Emph("turso db shell "+name))
		fmt.Printf("To see information about the database, including a connection URL, run:\n\n")
		fmt.Printf("   %s\n\n", internal.Emph("turso db show "+name))
		fmt.Printf("To get an authentication token for the database, run:\n\n")
		fmt.Printf("   %s\n\n", internal.Emph("turso db tokens create "+name))
		invalidateDatabasesCache()
		return nil
	},
}

func getDatabaseName(args []string) (string, error) {
	if len(args) > 0 && len(args[0]) > 0 {
		return args[0], nil
	}

	rng, err := codename.DefaultRNG()
	if err != nil {
		return "", err
	}
	return codename.Generate(rng, 0), nil
}

func groupFromFlag(client *turso.Client) (string, error) {
	groups, err := getGroups(client)
	if err != nil {
		return "", err
	}

	if groupFlag != "" {
		if !groupExists(groups, groupFlag) {
			return "", fmt.Errorf("group %s does not exist", groupFlag)
		}
		return groupFlag, nil
	}

	switch {
	case len(groups) == 0:
		return "default", nil
	case len(groups) == 1:
		return groups[0].Name, nil
	default:
		return "", fmt.Errorf("you have more than one database group. Please specify one with %s", internal.Emph("--group"))

	}
}

func groupExists(groups []turso.Group, name string) bool {
	for _, group := range groups {
		if group.Name == name {
			return true
		}
	}
	return false
}

func locationFromFlag(client *turso.Client) (string, error) {
	loc := locationFlag
	if loc == "" {
		loc, _ = closestLocation(client)
	}
	if !isValidLocation(client, loc) {
		return "", fmt.Errorf("location '%s' is not valid", loc)
	}
	return loc, nil
}

func shouldCreateGroup(client *turso.Client, name, location string) (bool, error) {
	groups, err := getGroups(client)
	if err != nil {
		return false, err
	}
	// we only create the default group automatically
	return name == "default" && len(groups) == 0, nil
}
