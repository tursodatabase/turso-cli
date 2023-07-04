package flags

import (
	"github.com/spf13/cobra"
)

var debugFlag bool

func AddDebugFlag(cmd *cobra.Command) {
	usage := "If set, shows dumps of all outgoing HTTP requests."
	cmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, usage)
	cmd.PersistentFlags().MarkHidden("debug")
}

func Debug() bool {
	return debugFlag
}
