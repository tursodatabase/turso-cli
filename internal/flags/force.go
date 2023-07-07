package flags

import (
	"github.com/spf13/cobra"
)

var overrideConfig bool

func AddOverrideConfigFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&overrideConfig, "override-config", false, "")
	cmd.PersistentFlags().MarkHidden("override-config")
}

func OverrideConfig() bool {
	return overrideConfig
}
