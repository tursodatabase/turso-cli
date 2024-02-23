package flags

import "github.com/spf13/cobra"

var attachFlag []string

func AddAttachClaims(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&attachFlag, "attach", nil, "list of database names with attach claim to be added to the token")
}

func AttachClaims() []string {
	return attachFlag
}
