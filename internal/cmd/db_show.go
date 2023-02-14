package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show database_name",
	Short: "Show information from a database.",
	Args: cobra.MatchAll(
		cobra.ExactArgs(1),
		dbNameValidator(0),
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		if db.Type != "logical" {
			return fmt.Errorf("only new databases, of type 'logical', support the show operation")
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		if showUrlFlag {
			fmt.Println(getDatabaseUrl(config, db))
			return nil
		}

		instances, err := client.Instances.List(db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		regions := make([]string, len(db.Regions))
		copy(regions, db.Regions)
		sort.Strings(regions)

		fmt.Println("Name:    ", db.Name)
		fmt.Println("URL:     ", getDatabaseUrl(config, db))
		fmt.Println("ID:      ", db.ID)
		fmt.Println("Regions: ", strings.Join(regions, ", "))
		fmt.Println()

		data := [][]string{}
		for _, instance := range instances {
			url := getInstanceUrl(config, db, instance)
			data = append(data, []string{instance.Name, instance.Type, instance.Region, url})
		}

		fmt.Print("Database Instances:\n")
		printTable([]string{"name", "type", "region", "url"}, data)

		return nil
	},
}
