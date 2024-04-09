package flags

import (
	"github.com/spf13/cobra"
)

var overagesFlag bool

func AddOverages(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&overagesFlag, "overages", false, "Enable or disable overages from the plan. If not selected, current overages configuration will not be changed.")
}

func Overages(cmd *cobra.Command) *bool {
	if !cmd.Flags().Changed("overages") {
		return nil
	}
	return &overagesFlag
}
