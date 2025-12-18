package cmd

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var multipartFlag bool

func init() {
	dbCmd.AddCommand(importCmd)
	addGroupFlag(importCmd)
	addRemoteEncryptionKeyFlag(importCmd)
	addRemoteEncryptionCipherFlag(importCmd)
	importCmd.Flags().BoolVar(&multipartFlag, "multipart", false, "force multipart upload")
}

var importCmd = &cobra.Command{
	Use:               "import [filename]",
	Short:             "Import a SQLite database file to Turso.",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: noFilesArg,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		if len(args) == 0 {
			return errors.New("filename is required: 'turso db import <filename>'")
		}
		filename := args[0]
		fromFileFlag = filename
		name := sanitizeDatabaseName(filename)
		return CreateDatabase(name)
	},
}

// Sanitize a SQLite database filename to be used as a cloud database name.
func sanitizeDatabaseName(filename string) string {
	base := filepath.Base(filename)

	extensions := []string{".db", ".sqlite", ".sqlite3", ".sl3", ".s3db", ".db3"}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.ToLower(base), ext) {
			return base[:len(base)-len(ext)]
		}
	}

	return base
}
