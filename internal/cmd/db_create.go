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
	addWaitFlag(createCmd, "Wait for the database to be ready to receive requests.")
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
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		region := locationFlag
		if region != "" && !isValidLocation(client, region) {
			return fmt.Errorf("location '%s' is not a valid one", region)
		}
		if region == "" {
			region, _ = closestLocation(client)
		}
		var image string
		if canary {
			image = "canary"
		} else {
			image = "latest"
		}
		start := time.Now()
		regionText := fmt.Sprintf("%s (%s)", locationDescription(client, region), region)

		dbFile, err := getDbFile(fromFileFlag)
		if err != nil {
			return err
		}

		dbText := ""
		if fromFileFlag != "" {
			dbText = fmt.Sprintf(" from file %s", internal.Emph(fromFileFlag))
		}

		description := fmt.Sprintf("Creating database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(regionText))
		spinner := prompt.Spinner(description)
		defer spinner.Stop()

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
			description = fmt.Sprintf("Uploading database file %s", internal.Emph(fromFileFlag))
			spinner.Text(description)

			err := client.Databases.Seed(name, dbFile)
			if err != nil {
				client.Databases.Delete(name)
				return fmt.Errorf("could not create database %s: %w", name, err)
			}

			description = fmt.Sprintf("Finishing to create database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(regionText))
			spinner.Text(description)
		}

		instance, err := client.Instances.Create(name, "", region, image)
		if err != nil {
			return err
		}

		if waitFlag || dbFile != nil {
			description = fmt.Sprintf("Waiting for database %s to be ready", internal.Emph(name))
			spinner.Text(description)
			if err = client.Instances.Wait(name, instance.Name); err != nil {
				return err
			}
		}

		spinner.Stop()
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

func getDbFile(path string) (*os.File, error) {
	if fromFileFlag == "" {
		return nil, nil
	}

	f, err := os.Open(fromFileFlag)
	if err != nil {
		return nil, fmt.Errorf("can't open %s: %w", fromFileFlag, err)
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("can't stat %s: %w", fromFileFlag, err)
	}

	if stat.Size() > (2 << 30) {
		return nil, fmt.Errorf("only files up to 2GiB are supported")
	}

	valid, err := isSQLiteFile(f)
	if err != nil {
		return nil, fmt.Errorf("error while reading %s: %w", fromFileFlag, err)
	}
	if !valid {
		return nil, fmt.Errorf("%s doesn't seem to be a SQLite file", fromFileFlag)
	}

	return f, nil
}
