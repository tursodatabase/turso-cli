package cmd

import (
	"fmt"
	"time"

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
	Args:              cobra.RangeArgs(2, 3),
	ValidArgsFunction: replicateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		if dbName == "" {
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
		client := createTursoClient()
		if !isValidRegion(client, region) {
			return fmt.Errorf("invalid region ID. Run %s to see a list of valid region IDs", turso.Emph("turso db regions"))
		}

		image := "latest"
		if canary {
			image = "canary"
		}

		original, err := getDatabase(client, dbName)
		if err != nil {
			return err
		}
		dbSettings := config.GetDatabaseSettings(original.ID)
		password := dbSettings.Password

		instanceName := ""
		if len(args) > 2 {
			instanceName = args[2]
		}

		regionText := fmt.Sprintf("%s (%s)", toLocation(client, region), region)
		s := startLoadingBar(fmt.Sprintf("Replicating database %s to %s ", turso.Emph(dbName), turso.Emph(regionText)))
		start := time.Now()
		instance, err := client.Instances.Create(dbName, instanceName, password, region, image)
		s.Stop()
		if err != nil {
			return fmt.Errorf("failed to create database: %s", err)
		}
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", turso.Emph(dbName), turso.Emph(regionText), int(elapsed.Seconds()))

		fmt.Printf("URL:\n\n")
		dbUrl := getInstanceUrl(config, &original, instance)
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", dbUrl)
		return nil
	},
}
