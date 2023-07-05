package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showUrlFlag, "url", false, "Show URL for the database HTTP API.")
	showCmd.Flags().BoolVar(&showInstanceUrlsFlag, "instance-urls", false, "Show URL for the HTTP API of all existing instances")
	showCmd.Flags().StringVar(&showInstanceUrlFlag, "instance-url", "", "Show URL for the HTTP API of a selected instance of a database. Instance is selected by instance name.")
	showCmd.RegisterFlagCompletionFunc("instance-url", completeInstanceName)
	showCmd.RegisterFlagCompletionFunc("instance-ws-url", completeInstanceName)
}

var showCmd = &cobra.Command{
	Use:               "show database_name",
	Short:             "Show information from a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		if showUrlFlag {
			fmt.Println(getDatabaseUrl(&db))
			return nil
		}

		if showHttpUrlFlag {
			fmt.Println(getDatabaseHttpUrl(&db))
			return nil
		}

		instances, usages, err := instancesAndUsage(client, db.Name)
		if err != nil {
			return fmt.Errorf("could not get instances of database %s: %w", db.Name, err)
		}

		if showInstanceUrlFlag != "" {
			for _, instance := range instances {
				if instance.Name == showInstanceUrlFlag {
					fmt.Println(getInstanceUrl(&db, &instance))
					return nil
				}
			}
			return fmt.Errorf("instance %s was not found for database %s. List known instances using %s", internal.Emph(showInstanceUrlFlag), internal.Emph(db.Name), internal.Emph("turso db show "+db.Name))
		}

		regions := make([]string, len(db.Regions))
		copy(regions, db.Regions)
		sort.Strings(regions)

		headers := []string{"Name", "Type", "Location"}
		if showInstanceUrlsFlag {
			headers = append(headers, "URL")
		}

		data := [][]string{}
		for _, instance := range instances {
			row := []string{instance.Name, instance.Type, instance.Region}
			if showInstanceUrlsFlag {
				url := getInstanceUrl(&db, &instance)
				row = append(row, url)
			}
			data = append(data, row)
		}

		fmt.Println("Name:          ", db.Name)
		fmt.Println("URL:           ", getDatabaseUrl(&db))
		fmt.Println("ID:            ", db.ID)
		fmt.Println("Locations:     ", strings.Join(regions, ", "))
		fmt.Println("Size:          ", humanize.Bytes(usages.Total.StorageBytesUsed))
		fmt.Println()
		fmt.Print("Database Instances:\n")
		printTable(headers, data)

		return nil
	},
}
