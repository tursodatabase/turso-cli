package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var withMetadata bool
var overwriteExport bool
var outputFile string

var exportCmd = &cobra.Command{
	Use:   "export <database>",
	Short: "Export a database snapshot and WAL from Turso to SQLite files.",
	Long: `Export a database snapshot and WAL from Turso to SQLite files.

This command exports a snapshot of the current generation of a Turso database
to a local SQLite file, along with any WAL (Write-Ahead Log) frames. The WAL
file will be saved as <database>.db-wal alongside the main database file.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		dbName := args[0]
		if outputFile == "" {
			outputFile = dbName + ".db"
		}
		err := ExportDatabase(dbName, outputFile, withMetadata, overwriteExport)
		if err != nil {
			return fmt.Errorf("failed to export database: %w", err)
		}
		fmt.Printf("Exported database to %s\n", outputFile)
		return nil
	},
}

func ExportDatabase(dbName, outputFile string, withMetadata bool, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(outputFile); err == nil {
			return fmt.Errorf("file %s already exists, use `--overwrite` flag to overwrite it", outputFile)
		}
	}
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	db, err := getDatabase(client, dbName)
	if err != nil {
		return fmt.Errorf("failed to find database: %w", err)
	}
	dbUrl := getDatabaseHttpUrl(&db)
	err = client.Databases.Export(dbName, dbUrl, outputFile, withMetadata, overwrite, remoteEncryptionKeyFlag())
	if err != nil {
		return err
	}
	return nil
}

func init() {
	exportCmd.Flags().BoolVar(&withMetadata, "with-metadata", false, "Include metadata in the export.")
	exportCmd.Flags().BoolVar(&overwriteExport, "overwrite", false, "Overwrite output file if it exists.")
	exportCmd.Flags().StringVar(&outputFile, "output-file", "", "Specify the output file name (default: <database>.db)")
	addRemoteEncryptionKeyFlag(exportCmd)
	dbCmd.AddCommand(exportCmd)
}
