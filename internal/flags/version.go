package flags

import (
	"github.com/spf13/cobra"
)

var versionFlag string

// AddVersion registers --version. Beyond "latest", the API accepts a specific
// turso-server version tag (e.g. "v0.34.2") from internal Turso users: the
// value is stamped as the turso.io/server-image marker on the group's
// provision request, so the group — and every database in it — is placed on a
// db-api fleet running exactly that version (composed from a matching dynamic
// fleet template). Validation is server-side; the CLI passes the value
// through untouched.
func AddVersion(cmd *cobra.Command, desc string) {
	cmd.Flags().StringVar(&versionFlag, "version", "", desc)
	_ = cmd.RegisterFlagCompletionFunc("version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"latest"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func Version() string {
	return versionFlag
}
