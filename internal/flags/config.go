package flags

import (
	"github.com/spf13/cobra"
)

var resetConfig bool

func AddResetConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&resetConfig, "reset-config", false, "")
	cmd.PersistentFlags().MarkHidden("reset-config")
}

func ResetConfig() bool {
	return resetConfig
}
