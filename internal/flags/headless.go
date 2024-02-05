package flags

import (
	"github.com/spf13/cobra"
)

var headless bool

func AddHeadless(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&headless, "headless", false, "Give users a link to start the process by themselves. Useful when the CLI can't interact with a web browser.")
}

func Headless() bool {
	return headless
}
