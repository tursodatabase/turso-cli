package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tursodatabase/turso-cli/internal"
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

func semverCompare(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	// fall back to lexicographic comparison if both don't have 3 parts
	if len(partsA) != 3 || len(partsB) != 3 {
		return strings.Compare(a, b)
	}

	// compare each part, and again fall back to lexicographic comparison if any part is not an integer
	for i := range 3 {
		numA, err := strconv.Atoi(partsA[i])
		if err != nil {
			return strings.Compare(a, b)
		}
		numB, err := strconv.Atoi(partsB[i])
		if err != nil {
			return strings.Compare(a, b)
		}
		if numA < numB {
			return -1
		} else if numA > numB {
			return 1
		}
	}
	return 0
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
		} else if semverCompare(version, latest) >= 0 {
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
