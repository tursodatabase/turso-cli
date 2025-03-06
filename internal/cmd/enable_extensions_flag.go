package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var enableExtensionsFlag bool

func addEnableExtensionsFlag(cmd *cobra.Command) {
	usage := []string{
		"Enables experimental support for SQLite extensions.",
	}
	cmd.Flags().BoolVar(&enableExtensionsFlag, "enable-extensions", false, strings.Join(usage, "\n"))
}
