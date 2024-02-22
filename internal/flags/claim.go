package flags

import "github.com/spf13/cobra"

var claimFlag []string

func AddClaim(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&claimFlag, "claim", nil, "Custom claims in JSON format")
}

func Claim() []string {
	return claimFlag
}
