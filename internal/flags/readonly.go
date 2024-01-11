package flags

import (
	"github.com/spf13/cobra"
)

var readOnlyFlag bool

func AddReadOnly(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&readOnlyFlag, "read-only", "r", false, "Token with read-only access")
}

func ReadOnly() bool {
	return readOnlyFlag
}
