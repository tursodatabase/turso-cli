package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/schollz/sqlite3dump"
	"github.com/spf13/cobra"
)

var groupFlag string

func addGroupFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&groupFlag, "group", "", "create the database in the specified group")
	cmd.RegisterFlagCompletionFunc("group", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		client, err := createTursoClientFromAccessToken(false)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		groups, _ := groupNames(client)
		return groups, cobra.ShellCompDirectiveNoFileComp
	})
}

var fromDBFlag string
var timestampFlag string

func addFromDBFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromDBFlag, "from-db", "", "Select another database to copy data from. To use data from a past version of the selected database, see the 'timestamp' flag.")
	cmd.RegisterFlagCompletionFunc("from-db", dbNameArg)
	cmd.Flags().StringVar(&timestampFlag, "timestamp", "", "Set a point in time in the past to copy data from the selected database. Must be used with the 'from-db' flag. Must be in RFC3339 format like '2023-09-29T10:16:13-03:00'")
}

func parseTimestampFlag() (*time.Time, error) {
	if timestampFlag == "" {
		return nil, nil
	}
	if fromDBFlag == "" {
		return nil, fmt.Errorf("--timestamp cannot be used without specifying --from-db")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampFlag)
	if err != nil {
		return nil, fmt.Errorf("provided timestamp was not in RFC3339 format like '2023-09-29T10:16:13-03:00'")
	}
	return &timestamp, nil
}

func parseDBSeedFlags(client *turso.Client) (*turso.DBSeed, error) {
	if countFlags(fromDBFlag, fromDumpFlag, fromFileFlag) > 1 {
		return nil, fmt.Errorf("only one of --from-db, --from-dump, or --from-file can be used at a time")
	}

	timestamp, err := parseTimestampFlag()
	if err != nil {
		return nil, err
	}

	if fromDBFlag != "" {
		return &turso.DBSeed{Type: "database", Name: fromDBFlag, Timestamp: timestamp}, nil
	}

	if fromFileFlag != "" {
		return handleDBFile(client, fromFileFlag)
	}

	if fromDumpFlag != "" {
		return handleDumpFile(client, fromDumpFlag)
	}

	return nil, nil
}

func handleDumpFile(client *turso.Client, file string) (*turso.DBSeed, error) {
	dump, err := validateDumpFile(file)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	spinner := prompt.Spinner("Uploading data...")
	defer spinner.Stop()

	dumpURL, err := client.Databases.UploadDump(dump)
	if err != nil {
		return nil, fmt.Errorf("could not upload dump: %w", err)
	}

	spinner.Stop()
	elapsed := time.Since(start)
	fmt.Printf("Uploaded data in %d seconds.\n\n", int(elapsed.Seconds()))

	return &turso.DBSeed{
		Type: "dump",
		URL:  dumpURL,
	}, nil
}

func validateDumpFile(name string) (*os.File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", name, err)
	}
	fileStat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file %s: %w", name, err)
	}
	if fileStat.Size() == 0 {
		return nil, fmt.Errorf("dump file is empty")
	}
	if fileStat.Size() > MaxDumpFileSizeBytes {
		return nil, fmt.Errorf("dump file is too large. max allowed size is 2GB")
	}
	return file, nil
}

func countFlags(flags ...string) (count int) {
	for _, flag := range flags {
		if flag != "" {
			count++
		}
	}
	return
}

func handleDBFile(client *turso.Client, file string) (*turso.DBSeed, error) {
	f, err := os.CreateTemp("", "")
	if err != nil {
		return nil, fmt.Errorf("could not create temporary file to dump database file: %w", err)
	}
	if err := sqlite3dump.Dump(file, f); err != nil {
		return nil, fmt.Errorf("could not dump database file: %w", err)
	}

	return handleDumpFile(client, f.Name())
}
