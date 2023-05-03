package cmd

import (
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(replicateCmd)
	addCanaryFlag(replicateCmd)
}

func replicateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := createTursoClientFromAccessToken(false)
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
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		if !isValidRegion(client, region) {
			return fmt.Errorf("invalid location ID. Run %s to see a list of valid location IDs", internal.Emph("turso db locations"))
		}

		image := "latest"
		if canary {
			image = "canary"
		}

		database, err := getDatabase(client, dbName)
		if err != nil {
			return err
		}

		instanceName := ""
		if len(args) > 2 {
			instanceName = args[2]
		}

		regionText := fmt.Sprintf("%s (%s)", toLocation(client, region), region)
		s := prompt.Spinner(fmt.Sprintf("Replicating database %s to %s ", internal.Emph(dbName), internal.Emph(regionText)))
		start := time.Now()
		_, err = client.Instances.Create(dbName, instanceName, region, image)
		s.Stop()
		if err != nil {
			return fmt.Errorf("failed to create database: %s", err)
		}
		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", internal.Emph(dbName), internal.Emph(regionText), int(elapsed.Seconds()))

		dbUrl := getDatabaseUrl(config, &database, false)
		showCmd := fmt.Sprintf("turso db show %s", dbName)
		fmt.Printf("Client connections using %s that are physically close to %s will now be routed to this replica for lower latency.\n\n", internal.Emph(dbUrl), internal.Emph(region))
		fmt.Printf("To see information about the database, including a connection URL specific to this location, run:\n\n\t%s\n", internal.Emph(showCmd))

		firstTime := config.RegisterUse("db_replicate")
		if firstTime {
			fmt.Println("How is your experience going? We'd love to know!")
			fmt.Printf("🗓   Book a call with us! You can do it with:\n\n\t%s\n", internal.Emph("turso account bookmeeting"))
			fmt.Printf("🎤   Or just send us your feedback:\n\n\t%s\n", internal.Emph("turso account feedback"))
		}

		return nil
	},
}
