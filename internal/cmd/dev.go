package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

func init() {
	rootCmd.AddCommand(devCmd)
	addDevPortFlag(devCmd)
	addDevFileFlag(devCmd)
	addDevSqldVersionFlag(devCmd)
	addAuthJwtFileFlag(devCmd)
}

var devCmd = &cobra.Command{
	Use:               "dev",
	Short:             "starts a local development server for Turso",
	Long:              fmt.Sprintf("starts a local development server for Turso.\n\nIf you're using a libSQL client SDK that supports SQLite database files on the local filesystem, then you might not need this server at all.\nInstead, you can use a %s URL with the path to the file you want the SDK to read and write.", internal.Emph("file:")),
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sqldNotFoundErr := fmt.Sprintf("%s.\nTo install it, follow the instructions at %s\nAlso make sure %s is on your PATH\n", internal.Warn("Could not start libsql-server"),
			internal.Emph("https://github.com/tursodatabase/libsql/blob/main/docs/BUILD-RUN.md"),
			internal.Emph("sqld"))

		version, err := getSqldVersion()
		if err != nil {
			fmt.Fprint(os.Stderr, sqldNotFoundErr)
			return err
		}
		if sqldVersion {
			fmt.Println(version)
			return nil
		}

		tempDir, err := os.MkdirTemp("", "*tursodev")
		if err != nil {
			return fmt.Errorf("Error creating temporary directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		if err := os.MkdirAll(filepath.Join(tempDir, "dbs"), 0o755); err != nil {
			return fmt.Errorf("Error creating directory: %w", err)
		}
		if err = os.Symlink(tempDir, filepath.Join(tempDir, "dbs", "default")); err != nil {
			return fmt.Errorf("Error creating link to file: %w", err)
		}

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
			if err := os.WriteFile(filepath.Join(tempDir, ".version"), []byte(extractSemver(version)), 0644); err != nil {
				return fmt.Errorf("Error writing version file: %w", err)
			}
		}

		addr := fmt.Sprintf("0.0.0.0:%d", devPort)
		conn := fmt.Sprintf("http://127.0.0.1:%d", devPort)

		sqldFlags := []string{
			"--no-welcome",
			"--http-listen-addr",
			addr,
			"-d",
			tempDir,
		}

		if authJwtFile != "" {
			sqldFlags = append(sqldFlags, "--auth-jwt-key-file", authJwtFile)
		}

		sqld := exec.Command("sqld", sqldFlags...)
		sqld.Env = append(os.Environ(), "RUST_LOG=error")

		// Set the appropriate output and error streams for the server process
		sqld.Stdout = os.Stdout
		sqld.Stderr = os.Stderr

		// Start the server process.
		err = sqld.Start()
		if err != nil {
			fmt.Fprint(os.Stderr, sqldNotFoundErr)
			return err
		}

		// Check if the server is actually running.
		maxAttempts := 3
		for i := range maxAttempts {
			_, err := http.Get(conn)
			if err == nil {
				break
			}
			if i == maxAttempts-1 {
				fmt.Fprintf(os.Stderr, "sqld not ready after %d health check attempts\n", maxAttempts)
				return err
			}
			time.Sleep(500 * time.Millisecond)
		}

		fmt.Printf("sqld listening on port %s.\n", internal.Emph(devPort))

		fmt.Printf("Use the following URL to configure your libSQL client SDK for local development:\n\n    %s\n\n",
			internal.Emph(conn))
		if authJwtFile != "" {
			fmt.Printf("Using auth token from file %s.\n\n", authJwtFile)
		} else {
			fmt.Printf("By default, no auth token is required when sqld is running locally. If you want to require authentication, use %s to specify a file containing the JWT key.\n\n", internal.Emph("--auth-jwt-key-file"))
		}
		if devFile != "" {
			fmt.Printf("Using database file %s.\n", internal.Emph(devFile))
		} else {
			fmt.Printf("This server is using an ephemeral database. Changes will be lost when this server stops.\nIf you want to persist changes, use %s to specify a SQLite database file instead.\n", internal.Emph("--db-file"))
		}

		waitCh := make(chan error, 1)
		go func() { waitCh <- sqld.Wait() }()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		select {
		case <-sigCh:
		case err = <-waitCh:
			return fmt.Errorf("sqld exited unexpectedly: %w", err)
		}

		// Terminate the server process
		err = sqld.Process.Kill()
		if err != nil {
			return fmt.Errorf("could not kill sqld: %w", err)
		}

		// Wait for the server process to exit.
		err = <-waitCh
		if err != nil {
			return fmt.Errorf("could not kill sqld: %w", err)
		}

		return nil
	},
}

func extractSemver(version string) string {
	regex := regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)
	return regex.FindString(version)
}
