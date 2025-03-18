package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(importCmd)
	addGroupFlag(importCmd)
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
			return fmt.Errorf("filename is required: 'turso db import <filename>'")
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
	return strings.TrimSuffix(base, ".db")
}
