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

		locationId := locationFlag
		if locationId == "" {
			locationId, _ = closestLocation(client)
		}
		if !isValidLocation(client, locationId) {
			return fmt.Errorf("location '%s' is not a valid one", locationId)
		}

		locationText := fmt.Sprintf("%s (%s)", locationDescription(client, locationId), locationId)
		start := time.Now()
		description := fmt.Sprintf("Creating database %s in %s", internal.Emph(name), internal.Emph(locationText))
		spinner := prompt.Spinner(description)
		defer spinner.Stop()
		if _, err = client.Databases.Create(name, locationId, "", "", groupFlag, fromDBFlag); err != nil {
			return fmt.Errorf("could not create database %s: %w", name, err)
		}

		spinner.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Created database %s in %s group in %d seconds.\n\n", internal.Emph(name), internal.Emph(groupFlag), int(elapsed.Seconds()))

		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", name)
		fmt.Printf("To see information about the database, including a connection URL, run:\n\n")
		fmt.Printf("   turso db show %s\n\n", name)
		fmt.Printf("To get an authentication token for the database, run:\n\n")
		fmt.Printf("   turso db tokens create %s\n\n", name)
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
