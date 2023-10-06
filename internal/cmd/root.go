package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal/flags"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func init() {
	rootCmd.PersistentFlags().StringP("config-path", "c", "", "Path to the directory with config file")
	if err := viper.BindPFlag("config-path", rootCmd.PersistentFlags().Lookup("config-path")); err != nil {
		fmt.Fprintf(os.Stderr, "error binding token flag: %s", err)
	}
	rootCmd.PersistentFlags().BoolVar(&noMultipleTokenSourcesWarning, "no-multiple-token-sources-warning", false, "Don't warn about multiple access token sources")

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		settings.PersistChanges()
		settings, err := settings.ReadSettings()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read settings: %s\n", err)
			os.Exit(1)
		}
		if settings.GetAutoupdate() == "on" && time.Now().Unix() >= settings.GetLastUpdateCheck()+int64(24*60*60) {
			fmt.Println("Checking for updates...")
			latest, err := fetchLatestVersion()
			settings.SetLastUpdateCheck(time.Now().Unix())
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to get latest version: %s\n", err)
				return
			}

			parsedVersion, _ := semver.NewVersion(version)
			parsedLatest, _ := semver.NewVersion(latest)

			if version != "dev" && parsedVersion.LessThan(parsedLatest) {
				Update()
				return
			}
			fmt.Println("You're already on the latest version")
		}
	}
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	flags.AddDebugFlag(rootCmd)
	flags.AddResetConfigFlag(rootCmd)
}
