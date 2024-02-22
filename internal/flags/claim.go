package flags

import "github.com/spf13/cobra"

var attachFlag []string

func AddAttachClaim(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&attachFlag, "attach", nil, "list of database names with attach claim to be added to the token")
}

func AttachClaim() []string {
	return attachFlag
}
