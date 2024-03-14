package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

func init() {
	dbCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&showUrlFlag, "url", false, "Show URL for the database HTTP API.")
	showCmd.Flags().BoolVar(&showHttpUrlFlag, "http-url", false, "Show HTTP URL for the database HTTP API.")
	showCmd.Flags().BoolVar(&showInstanceUrlsFlag, "instance-urls", false, "Show URL for the HTTP API of all existing instances")
	showCmd.Flags().StringVar(&showInstanceUrlFlag, "instance-url", "", "Show URL for the HTTP API of a selected instance of a database. Instance is selected by instance name.")
	showCmd.RegisterFlagCompletionFunc("instance-url", completeInstanceName)
	showCmd.RegisterFlagCompletionFunc("instance-ws-url", completeInstanceName)
}

var showCmd = &cobra.Command{
	Use:               "show <database-name>",
	Short:             "Show information from a database.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		db, err := getDatabase(client, args[0], true)
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

		instances, dbUsage, err := instancesAndUsage(client, db.Name)
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
		if db.Group != "" {
			fmt.Println("Group:         ", db.Group)
		}
		if db.Version != "" {
			fmt.Println("Version:       ", db.Version)
		}
		fmt.Println("Locations:     ", strings.Join(regions, ", "))
		fmt.Println("Size:          ", humanize.Bytes(dbUsage.Usage.StorageBytesUsed))
		fmt.Println("Sleeping:      ", formatBool(db.Sleeping))
		fmt.Println("Bytes Synced:  ", humanize.Bytes(dbUsage.Usage.BytesSynced))

		fmt.Println()

		if len(instances) == 0 {
			fmt.Printf("🛠 Run %s to finish your database creation!\n", internal.Emph("turso db replicate "+db.Name))
			return nil
		}

		fmt.Print("Database Instances:\n")
		printTable(headers, data)

		return nil
	},
}
