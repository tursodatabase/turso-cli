package flags

import (
	"github.com/spf13/cobra"
)

var all bool

func AddAll(cmd *cobra.Command, usage string) {
	cmd.Flags().BoolVarP(&all, "all", "a", false, usage)
}

func All() bool {
	return all
}
