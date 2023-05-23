package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/athoscouto/codename"
	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt/spinner"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
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
		name, err := getDatabaseName(args)
		if err != nil {
			return err
		}

		if !turso.IsValidName(name) {
			return errors.New("invalid name: names only support lowercase letters, numbers, and hyphens")
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		region := locationFlag
		if region == "" {
			region, _ = closestLocation(client)
		}
		if !isValidLocation(client, region) {
			return fmt.Errorf("location '%s' is not a valid one", region)
		}

		image := "latest"
		if canary {
			image = "canary"
		}

		dbFile, err := getDbFile(fromFileFlag)
		if err != nil {
			return err
		}

		dbText := ""
		if fromFileFlag != "" {
			dbText = fmt.Sprintf(" from file %s", internal.Emph(fromFileFlag))
		}
		regionText := fmt.Sprintf("%s (%s)", locationDescription(client, region), region)

		start := time.Now()
		description := fmt.Sprintf("Creating database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(regionText))
		spinner := spinner.Start(description)
		defer spinner.Stop()

		if _, err := client.Databases.Create(name, region, image); err != nil {
			return fmt.Errorf("could not create database %s: %w", name, err)
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
