package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/athoscouto/codename"
	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(createCmd)
	addCanaryFlag(createCmd)
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
		client, err := createTursoClient()
		if err != nil {
			return err
		}
		region := locationFlag
		if region != "" && !isValidRegion(client, region) {
			return fmt.Errorf("location '%s' is not a valid one", region)
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

		var dbText string
		if dbFromFile != "" {
			dbText = fmt.Sprintf(" from file %s", internal.Emph(dbFromFile))
		} else {
			dbText = ""
		}

		description := fmt.Sprintf("Creating database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(regionText))
		bar := prompt.Spinner(description)
		defer bar.Stop()

		// Do this before database creation, so we are able to throw those errors
		// early, before we call fly
		var dbFile *os.File
		if dbFromFile != "" {
			f, err := os.Open(dbFromFile)
			if err != nil {
				return fmt.Errorf("can't open %s: %w", dbFromFile, err)
			}

			stat, err := f.Stat()
			if err != nil {
				return fmt.Errorf("can't stat %s: %w", dbFromFile, err)
			}

			if stat.Size() > (128 << 10) {
				return fmt.Errorf("only files up to 128MiB are supported")
			}

			valid, err := isSQLiteFile(f)
			if err != nil {
				return fmt.Errorf("error while reading %s: %w", dbFromFile, err)
			}
			if !valid {
				return fmt.Errorf("%s doesn't seem to be a SQLite file", dbFromFile)
			}
			dbFile = f
		}

		res, err := client.Databases.Create(name, region, image)
		if err != nil {
			return fmt.Errorf("could not create database %s: %w", name, err)
		}
		dbSettings := settings.DatabaseSettings{
			Name:     res.Database.Name,
			Username: res.Username,
			Password: res.Password,
		}

		if dbFile != nil {
			defer dbFile.Close()
			err := client.Databases.Seed(name, dbFile)
			if err != nil {
				client.Databases.Delete(name)
				return fmt.Errorf("could not create database %s: %w", name, err)
			}
		}

		if _, err = client.Instances.Create(name, "", res.Password, region, image); err != nil {
			return err
		}

		bar.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Created database %s in %s in %d seconds.\n\n", internal.Emph(name), internal.Emph(regionText), int(elapsed.Seconds()))

		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", name)
		fmt.Printf("To see information about the database, including a connection URL, run:\n\n")
		fmt.Printf("   turso db show %s\n\n", name)
		config.AddDatabase(res.Database.ID, &dbSettings)
		config.InvalidateDbNamesCache()
		firstTime := config.RegisterUse("db_create")
		if firstTime {
			fmt.Printf("✏️  Now that you created a database, the next step is to create a replica. Why don't we try?\n\t%s\n\t%s\n",
				internal.Emph("turso db locations"), internal.Emph(fmt.Sprintf("turso db replicate %s [location]", name)))
		}
		return nil
	},
}
