package cmd

import (
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
		return "", err
	}
	return string(version), nil
}
