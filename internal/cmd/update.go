package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
)

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(autoupdateCmd)
	autoupdateCmd.AddCommand(autoupdateEnableCmd)
	autoupdateCmd.AddCommand(autoupdateDisableCmd)
}

func IsUnderHomebrew() bool {
	binary, err := os.Executable()
	if err != nil {
		return false
	}

	brewExe, err := exec.LookPath("brew")
	if err != nil {
		return false
	}

	brewPrefixBytes, err := exec.Command(brewExe, "--prefix").Output()
	if err != nil {
		return false
	}

	brewBinPrefix := filepath.Join(strings.TrimSpace(string(brewPrefixBytes)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(binary, brewBinPrefix)
}

var autoupdateCmd = &cobra.Command{
	Use:   "autoupdate",
	Short: "Manage your CLI autoupdate settings",
}

var autoupdateEnableCmd = &cobra.Command{
	Use:               "enable",
	Short:             "Enable autoupdates for the CLI",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("failed to read settings: %w", err)
		}
		settings.SetAutoupdate(true)
		settings.SetLastUpdateCheck(time.Now().Unix())
		fmt.Println("Autoupdates enabled")
		return nil
	},
}

var autoupdateDisableCmd = &cobra.Command{
	Use:               "disable",
	Short:             "Disable autoupdates for the CLI",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("failed to read settings: %w", err)
		}
		settings.SetAutoupdate(false)
		fmt.Println("Autoupdates disabled")
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:               "update",
	Short:             "Update the CLI to the latest version",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		latest, err := fetchLatestVersion()
		if err != nil {
			return fmt.Errorf("failed to get version information: %w", err)
		}

		fmt.Printf("Current version: %s, latest version: %s\n", version, latest)
		if version == "dev" {
			fmt.Println("You're compiling from source. How much more up2date do you want to be?")
			return nil
		} else if version >= latest {
			fmt.Printf("version %s is already latest\n", internal.Emph(version))
			return nil
		}

		return Update()
	},
}

func Update() error {
	var updateCmd string

	if IsUnderHomebrew() {
		updateCmd = "brew update && brew upgrade turso"
	} else {
		updateCmd = "curl -sSfL \"https://get.tur.so/install.sh\" | sh"
	}
	command := exec.Command("sh", "-c", updateCmd)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		return fmt.Errorf("failed to execute update command: %w", err)
	}
	return nil
}
