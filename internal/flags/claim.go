package flags

import "github.com/spf13/cobra"

var claimFlag []string

func AddClaim(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&claimFlag, "claim", nil, "list of database claims to be added to the token")
}

func Claim() []string {
	return claimFlag
}
