package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	"golang.org/x/exp/maps"

	"github.com/athoscouto/codename"
	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/manifoldco/promptui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(createCmd)
	addCanaryFlag(createCmd)
	addEnableExtensionsFlag(createCmd)
	addDbFromFileFlag(createCmd)
	addLocationFlag(createCmd, "Location ID. If no ID is specified, closest location to you is used by default.")
	addWaitFlag(createCmd, "Wait for the database to be ready to receive requests.")
}

func firstTimeHint(dbName string, image string, client *turso.Client, location string, locations map[string]string) {
	str := fmt.Sprintf("🎉 Congrats on creating your first database! Shall we make it available on another location?\n(Don't worry! We'll only ask you this the first time.).\n%s", internal.Emph("Let's do it?"))
	doReplica, err := promptConfirmation(str)
	fmt.Println("")
	if err != nil {
		doReplica = false
	}

	replicaStr := fmt.Sprintf("If you want to create a replica later, you can pick a location with %s, and then:\n\n   %s\n\n",
		internal.Emph("turso db locations"), internal.Emph(fmt.Sprintf("turso db replicate %s [location]", dbName)))

	switch doReplica {
	case true:
		suggestedLoc, suggestedLocationName := suggestedLocation(location, locations)

		ids := maps.Keys(locations)
		sort.Strings(ids)

		columns := make([]interface{}, 0)
		columns = append(columns, "ID↓")
		columns = append(columns, "LOCATION")

		tbl := turso.LocationsTable(columns)

		for _, id := range ids {
			if id == location {
				continue
			}
			var text string
			if id == suggestedLoc {
				text = fmt.Sprintf("%s [suggested]", locations[id])
				tbl.AddRow(internal.Emph(id), internal.Emph(text))
			} else {
				text = locations[id]
				tbl.AddRow(id, text)
			}
		}
		fmt.Printf("Great!! Where? We suggest %s, since you don't yet have coverage in %s\n", internal.Emph(suggestedLoc), internal.Emph(suggestedLocationName))
		tbl.Print()
		fmt.Printf("\n%s ", internal.Emph("Your choice"))
		var chosen string
		fmt.Scanf("%s", &chosen)
		if chosen == "" {
			fmt.Printf("Ok! %s", replicaStr)
		} else {
			if !isValidLocation(client, chosen) {
				fmt.Printf("invalid location ID. Skipping\n")
			} else {
				err = replicate(client, dbName, chosen, locations[chosen], image)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error while creating replica: %v", err)
				} else {
					fmt.Printf("Don't forget: %s", replicaStr)
				}
			}
		}
	case false:
		fmt.Printf("Ok! %s", replicaStr)
	}

}

func replicate(client *turso.Client, dbName string, location string, locationText string, image string) error {
	s := prompt.Spinner(fmt.Sprintf("Replicating database %s to %s ", internal.Emph(dbName), internal.Emph(locationText)))
	defer s.Stop()

	start := time.Now()
	instance, err := client.Instances.Create(dbName, "", location, image)
	if err != nil {
		return fmt.Errorf("failed to create database: %s", err)
	}

	if waitFlag {
		description := fmt.Sprintf("Waiting for replica of %s in %s to be ready", internal.Emph(dbName), internal.Emph(locationText))
		s.Text(description)
		if err = client.Instances.Wait(dbName, instance.Name); err != nil {
			return err
		}
	}

	s.Stop()
	end := time.Now()
	elapsed := end.Sub(start)
	fmt.Printf("Replicated database %s to %s in %d seconds.\n\n", internal.Emph(dbName), internal.Emph(locationText), int(elapsed.Seconds()))
	return nil
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

		config, err := settings.ReadSettings()
		if err != nil {
			return err
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

		locations, err := locations(client)
		if err != nil {
			return err
		}

		image := "latest"
		if canaryFlag {
			image = "canary"
		}

		extensions := ""
		if enableExtensionsFlag {
			extensions = "all"
		}

		dbFile, err := getDbFile(fromFileFlag)
		if err != nil {
			return err
		}

		dbText := ""
		if fromFileFlag != "" {
			dbText = fmt.Sprintf(" from file %s", internal.Emph(fromFileFlag))
		}
		locationText := fmt.Sprintf("%s (%s)", locationDescription(client, locationId), locationId)

		start := time.Now()
		description := fmt.Sprintf("Creating database %s%s in %s", internal.Emph(name), dbText, internal.Emph(locationText))
		spinner := prompt.Spinner(description)
		defer spinner.Stop()
		_, err = client.Databases.Create(name, locationId, image, extensions)
		if err != nil && err.Error() == "location error" {
			spinner.Stop()
			fmt.Printf("Region %s is currently experiencing issues. Please pick another from the 3 closest locations to %s or try again later\n", internal.Emph(locationId), internal.Emph(locationId))

			location, _ := client.Locations.GetLocation(locationId)

			closestLocationCodes := make([]string, 0)
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

			_, locationId, err = promptSelect.Run()
			if err != nil {
				fmt.Printf("Prompt failed %v\n", err)
				return nil
			}

			locationText = fmt.Sprintf("%s (%s)", locationDescription(client, locationId), locationId)

			description = fmt.Sprintf("Creating database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(locationText))
			spinner = prompt.Spinner(description)
			defer spinner.Stop()
			_, err = client.Databases.Create(name, locationId, image, extensions)

			if err != nil {
				return fmt.Errorf("Please retry later, location %s is also experiencing issues: %w", locationId, err)
			}
		}
		if err != nil {
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

			description = fmt.Sprintf("Finishing to create database %s%s in %s ", internal.Emph(name), dbText, internal.Emph(locationText))
			spinner.Text(description)
		}

		instance, err := client.Instances.Create(name, "", locationId, image)
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
		fmt.Printf("Created database %s in %s in %d seconds.\n\n", internal.Emph(name), internal.Emph(locationText), int(elapsed.Seconds()))

		firstTime := config.RegisterUse("db_create")
		isInteractive := isatty.IsTerminal(os.Stdin.Fd())
		isOnlyDatabase := false
		databases, err := client.Databases.List()
		if err == nil && len(databases) == 1 {
			isOnlyDatabase = true
		}

		if firstTime && isInteractive && isOnlyDatabase {
			firstTimeHint(name, image, client, locationId, locations)
		}

		fmt.Printf("You can start an interactive SQL shell with:\n\n")
		fmt.Printf("   turso db shell %s\n\n", name)
		fmt.Printf("To see information about the database, including a connection URL, run:\n\n")
		fmt.Printf("   turso db show %s\n\n", name)
		fmt.Printf("To get an authentication token for the database, run:\n\n")
		fmt.Printf("   turso db tokens create %s\n\n", name)

		config.InvalidateDatabasesCache()

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
