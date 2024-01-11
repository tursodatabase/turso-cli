package flags

import (
	"github.com/spf13/cobra"
)

var yesFlag bool

func AddYes(cmd *cobra.Command, desc string) {
	cmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the update of the group")
}

func Yes() bool {
	return yesFlag
}
