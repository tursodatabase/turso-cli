package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/settings"
)

var rootCmd = &cobra.Command{
	Use:     "turso",
	Version: version,
	Long:    "Turso CLI",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var noMultipleTokenSourcesWarning bool

func requiresLogin(cmd *cobra.Command) bool {
	path := cmd.CommandPath()
	allowlist := []string{
		"turso auth login",
		"turso auth signup",
		"turso config set token",
		"turso update",
		"turso completion",
		"turso db shell",
		"turso dev",
	}
	for _, allowed := range allowlist {
		if strings.HasPrefix(path, allowed) {
			return false
		}
	}
	return true
}

func init() {
	rootCmd.PersistentFlags().StringP("config-path", "c", "", "Path to the directory with config file")
	if err := viper.BindPFlag("config-path", rootCmd.PersistentFlags().Lookup("config-path")); err != nil {
		fmt.Fprintf(os.Stderr, "error binding token flag: %s", err)
	}
	rootCmd.PersistentFlags().BoolVar(&noMultipleTokenSourcesWarning, "no-multiple-token-sources-warning", false, "Don't warn about multiple access token sources")
	_ = rootCmd.PersistentFlags().MarkHidden("no-multiple-token-sources-warning")

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		configSettings, err := settings.ReadSettings()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading settings:", err)
			return
		}
		settings.PersistChanges()

		if !isInteractive() {
			return
		}

		if version == "dev" {
			return
		}
		if configSettings.GetAutoupdate() == "on" && time.Now().Unix() >= configSettings.GetLastUpdateCheck()+int64(24*60*60) {
			latest, err := fetchLatestVersion()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "Error fetching latest version:", err)
				return
			}

			parsedVersion, err := semver.NewVersion(version)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "Error parsing current version:", err)
				return
			}
			parsedLatest, err := semver.NewVersion(latest)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "Error parsing latest version:", err)
				return
			}

			if parsedVersion.LessThan(parsedLatest) {
				fmt.Println("Updating to the latest version")
				err := Update()
				if err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "Error updating:", err)
				}
				fmt.Printf("\nYou can disable automatic updates with %s\n", internal.Emph("turso config set autoupdate off"))
			}
			configSettings.SetLastUpdateCheck(time.Now().Unix())
			settings.PersistChanges()
		}
	}
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if !requiresLogin(cmd) {
			return
		}
		VerifyUserIsLoggedIn()
	}
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	flags.AddDebugFlag(rootCmd)
	flags.AddResetConfigFlag(rootCmd)
}

func VerifyUserIsLoggedIn() {
	_, err := getAccessToken()
	if errors.Is(err, ErrNotLoggedIn) {
		fmt.Printf("You are not logged in, please login with %s before running other commands.\n", internal.Emph("turso auth login"))
		os.Exit(0)
	}
}
