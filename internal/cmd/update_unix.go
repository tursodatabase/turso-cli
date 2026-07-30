//go:build !windows

package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

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
	if err := command.Run(); err != nil {
		return fmt.Errorf("failed to execute update command: %w", err)
	}
	return nil
}
