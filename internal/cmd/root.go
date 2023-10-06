package cmd

import (
	"fmt"
	"os"
	"time"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func init() {
	rootCmd.PersistentFlags().StringP("config-path", "c", "", "Path to the directory with config file")
	if err := viper.BindPFlag("config-path", rootCmd.PersistentFlags().Lookup("config-path")); err != nil {
		fmt.Fprintf(os.Stderr, "error binding token flag: %s", err)
	}
	rootCmd.PersistentFlags().BoolVar(&noMultipleTokenSourcesWarning, "no-multiple-token-sources-warning", false, "Don't warn about multiple access token sources")

	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		settings.PersistChanges()
		if version == "dev" {
			return
		}
		settings, _ := settings.ReadSettings()
		if settings.GetAutoupdate() == "on" && time.Now().Unix() >= settings.GetLastUpdateCheck()+int64(24*60*60) {
			latest, _ := fetchLatestVersion()
			settings.SetLastUpdateCheck(time.Now().Unix())

			parsedVersion, _ := semver.NewVersion(version)
			parsedLatest, _ := semver.NewVersion(latest)

			if parsedVersion.LessThan(parsedLatest) {
				fmt.Println("Updating to the latest version")
				Update()
				return
			}
		}
	}
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	flags.AddDebugFlag(rootCmd)
	flags.AddResetConfigFlag(rootCmd)
}
