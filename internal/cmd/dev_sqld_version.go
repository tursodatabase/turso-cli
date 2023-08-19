package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var sqldVersion bool

func addDevSqldVersionFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&sqldVersion, "version", "v", false, "sqld version")
}

func getSqldVersion() (string, error) {
	sqld := exec.Command("sqld", "--version")
	sqld.Env = append(os.Environ(), "RUST_LOG=error")
	version, err := sqld.Output()
	if err != nil {
		fmt.Println("Error running sqld --version:", err)
		return "", err
	}
	return string(version), nil
}
