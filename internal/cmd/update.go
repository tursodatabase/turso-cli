package cmd

import (
	"fmt"
	"strings"

	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/chiselstrike/iku-turso-cli/internal"
)

func init() {
	rootCmd.AddCommand(updateCmd)
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

var updateCmd = &cobra.Command{
	Use:               "update",
	Short:             "Update the CLI to the latest version",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		var updateCmd string
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

		if IsUnderHomebrew() {
			updateCmd = "brew upgrade turso"
		} else {
			updateCmd = "curl -sSfL \"https://get.tur.so/install.sh\" | sh"
		}
		command := exec.Command("sh", "-c", updateCmd)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		err = command.Run()
		if err != nil {
			return fmt.Errorf("failed to execute update command: %w", err)
		}
		return nil
	},
}
