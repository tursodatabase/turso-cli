package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(devCmd)
	addDevPortFlag(devCmd)
	addDevFileFlag(devCmd)
}

var devCmd = &cobra.Command{
	Use:               "dev",
	Short:             "starts a local development server for Turso",
	Long:              fmt.Sprintf("starts a local development server for Turso.\n\nIf you're using a libSQL client SDK that supports SQLite database files on the local filesystem, then you might not need this server at all.\nInstead, you can use a %s URL with the path to the file you want the SDK to read and write.", internal.Emph("file:")),
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		tempDir, err := os.MkdirTemp("", "*tursodev")
		if err != nil {
			return fmt.Errorf("Error creating temporary directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		if devFile != "" {
			absDevFile, err := filepath.Abs(devFile)
			if err != nil {
				return fmt.Errorf("Error getting absolute path: %w", err)
			}
			destFile := filepath.Join(tempDir, "data")
			err = os.Symlink(absDevFile, destFile)
			if err != nil {
				return fmt.Errorf("Error creating link to file: %w", err)
			}
		}

		addr := fmt.Sprintf("0.0.0.0:%d", devPort)
		conn := fmt.Sprintf("ws://127.0.0.1:%d", devPort)

		sqld := exec.Command("sqld", "--no-welcome", "--http-listen-addr", addr, "-d", tempDir)
		sqld.Env = append(os.Environ(), "RUST_LOG=error")

		// Set the appropriate output and error streams for the server process
		sqld.Stdout = os.Stdout
		sqld.Stderr = os.Stderr

		// Start the server process
		err = sqld.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s.\no install it, follow the instructions at %s.\nAlso make sure %s is on your PATH\n", internal.Warn("Could not start libsql-server"),
				internal.Emph("https://github.com/libsql/sqld/blob/main/docs/BUILD-RUN.md"),
				internal.Emph("sqld"))
			return err
		}
		fmt.Printf("%s sqld listening on port %s. Use this URL to configure your libSQL client SDK for local development: %s\n\n",
			internal.Emph("→  "), internal.Emph(devPort), internal.Emph(conn))

		if devFile != "" {
			fmt.Printf("Using database file %s.\n", internal.Emph(devFile))
		} else {
			fmt.Printf("This server is using an ephemeral database. Changes will be lost when this server stops. If you want to persist changes, use %s to specify a SQLite database file instead.\n", internal.Emph("--db-file"))
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		// Terminate the server process
		err = sqld.Process.Kill()
		if err != nil {
			return fmt.Errorf("could not kill sqld: %w", err)
		}
		return nil
	},
}
