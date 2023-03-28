package cmd

import (
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(replicateCmd)
	addCanaryFlag(replicateCmd)
}

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	if len(args) == 1 {
		return getRegionIds(client), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	return dbNameArg(cmd, args, toComplete)
}

var replicateCmd = &cobra.Command{
	Use:               "replicate database_name location_id",
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
			return fmt.Errorf("you must specify a database location ID to replicate it")
		}
		cmd.SilenceUsage = true
		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		client, err := createTursoClient()
		if err != nil {
			return err
		}
		if !isValidRegion(client, region) {
			return fmt.Errorf("invalid location ID. Run %s to see a list of valid location IDs", turso.Emph("turso db locations"))
		}

		image := "latest"
		if canary {
			image = "canary"
		}

		database, err := getDatabase(client, dbName)
		if err != nil {
			return err
		}
		dbSettings := config.GetDatabaseSettings(database.ID)
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
		dbUrl := getInstanceUrl(config, &database, instance)
		fmt.Printf("   %s\n\n", dbUrl)
		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", dbUrl)
		return nil
	},
}
