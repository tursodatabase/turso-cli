package cmd

import (
	_ "embed"
	"os"

	"github.com/spf13/cobra"
)

//go:generate sh -c "printf %s $(../../script/version.sh) > version.txt"
//go:embed version.txt
var version string

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
