package cmd

import (
	"fmt"
	"time"

	"github.com/athoscouto/codename"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:               "create [flags] [database_name]",
	Short:             "Create a database.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 0 || args[0] == "" {
			rng, err := codename.DefaultRNG()
			if err != nil {
				return err
			}
			name = codename.Generate(rng, 0)
		} else {
			name = args[0]
		}
		client := createTursoClient()
		region := region
		if region != "" && !isValidRegion(client, region) {
			return fmt.Errorf("region '%s' is not a valid one", region)
		}
		if region == "" {
			region = probeClosestRegion()
		}
		var image string
		if canary {
			image = "canary"
		} else {
			image = "latest"
		}
		start := time.Now()
		regionText := fmt.Sprintf("%s (%s)", toLocation(client, region), region)
		description := fmt.Sprintf("Creating database %s in %s ", turso.Emph(name), turso.Emph(regionText))
		bar := startLoadingBar(description)
		defer bar.Stop()
		res, err := client.Databases.Create(name, region, image)
		if err != nil {
			return fmt.Errorf("could not create database %s: %w", name, err)
		}
		dbSettings := settings.DatabaseSettings{
			Name:     res.Database.Name,
			Host:     res.Database.Hostname,
			Username: res.Username,
			Password: res.Password,
		}

		if _, err = client.Instances.Create(name, res.Password, region, image); err != nil {
			return err
		}

		bar.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Created database %s to %s in %d seconds.\n\n", turso.Emph(name), turso.Emph(regionText), int(elapsed.Seconds()))

		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", name)
		fmt.Printf("To obtain connection URL, run:\n\n")
		fmt.Printf("   turso db show --url %s\n\n", name)
		config.AddDatabase(res.Database.ID, &dbSettings)
		config.InvalidateDbNamesCache()
		return nil
	},
}
