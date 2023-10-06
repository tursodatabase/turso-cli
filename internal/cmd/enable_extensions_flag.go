package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

var enableExtensionsFlag bool

func addEnableExtensionsFlag(cmd *cobra.Command) {
	usage := []string{
		"Enables experimental support for SQLite extensions.",
		"If you would like to see some extension included, run " + internal.Emph("turso account feedback") + ".",
		internal.Warn("Warning") + ": extensions support is experimental and subject to change",
	}
	cmd.Flags().BoolVar(&enableExtensionsFlag, "enable-extensions", false, strings.Join(usage, "\n"))
}
