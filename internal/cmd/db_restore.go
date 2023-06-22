//go:build preview
// +build preview

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbRestoreCmd)
	addTimestampFlags(dbRestoreCmd)
	addOutputFlag(dbRestoreCmd)
}

var dbRestoreCmd = &cobra.Command{
	Use:               "restore {database_name} -o {restored_db}",
	Short:             "Restore database.",
	Example:           "turso db restore name-of-my-amazing-db -o path-to-my-amazing-db-dump.sql",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbName := args[0]
		if dbName == "" {
			return fmt.Errorf("please specify a database name")
		}

		timestamp, err := getTimestamp(cmd)
		if err != nil {
			return err
		}
		if outputFlag == "" {
			return fmt.Errorf("please specify an output file")
		}

		cmd.SilenceUsage = true

		outputAlreadyExists, err := checkIfFileExists(outputFlag)
		if err != nil {
			return fmt.Errorf("error checking if output file exists: %w", err)
		}

		if outputAlreadyExists {
			return fmt.Errorf("output file %s already exists", internal.Emph(outputFlag))
		}

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		spinner := prompt.Spinner(fmt.Sprintf("Starting restore for database %s", internal.Emph(dbName)))
		defer spinner.Stop()

		start := time.Now()

		restore, err := client.Restore.Create(dbName, timestamp)
		if err != nil {
			return err
		}

		for restore.State != turso.RestoreStateRestored {
			spinner.Text(fmt.Sprintf("Waiting restore for database %s to be finished", internal.Emph(dbName)))
			time.Sleep(1 * time.Second)
			restore, err = client.Restore.Get(dbName, restore.ID)
			if err != nil {
				return fmt.Errorf("failed to wait for restore: %w", err)
			}
		}

		spinner.Text(fmt.Sprintf("Downloading restore for database %s", internal.Emph(dbName)))
		fileReader, err := client.Restore.Download(dbName, restore.ID)
		if err != nil {
			return fmt.Errorf("failed to download restore: %w", err)
		}
		defer fileReader.Close()

		file, err := os.Create(outputFlag)
		if err != nil {
			return fmt.Errorf("error creating file: %v", err)
		}
		defer file.Close()

		_, err = io.Copy(file, fileReader)
		if err != nil {
			return fmt.Errorf("failed to write restore to file: %w", err)
		}

		end := time.Now()
		elapsed := end.Sub(start)
		fmt.Printf("Restored database %s to %s in %s.\n\n", internal.Emph(dbName), internal.Emph(outputFlag), elapsed.String())

		return nil
	},
}

func checkIfFileExists(filename string) (bool, error) {
	if _, err := os.Stat(filename); err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}
