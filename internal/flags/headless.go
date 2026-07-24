package flags

import (
	"github.com/spf13/cobra"
)

var headless bool

func AddHeadless(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&headless, "headless", false, "Print a login URL and prompt for a pasted access token instead of opening a browser callback. Useful on SSH, WSL, or hosts without a usable browser.")
}

func Headless() bool {
	return headless
}
