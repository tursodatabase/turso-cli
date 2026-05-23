package cmd

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	apiTokensCmd.AddCommand(listApiTokensCmd)
}

var listApiTokensCmd = &cobra.Command{
	Use:   "list",
	Short: "List API tokens.",
	Long: "" +
		"API tokens are revocable non-expiring tokens that authenticate holders as the user who minted them.\n" +
		"They can be used to implement automations with the " + internal.Emph("turso") + " CLI or the platform API.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		apiTokens, err := client.ApiTokens.List()
		if err != nil {
			return err
		}

		data := [][]string{}
		for _, apiToken := range apiTokens {
			data = append(data, []string{
				apiToken.Name,
				formatTokenOrgScope(apiToken),
				formatTokenPermissions(apiToken),
				apiToken.CreatedAt,
			})
		}
		printTable([]string{"Name", "Scope", "Permissions", "Created At"}, data)

		return nil
	},
}

// formatTokenOrgScope renders the token's binding for the "Scope" column:
// unrestricted tokens read "all", org-scoped tokens show the org slug, and
// group-scoped tokens show "<org>/<group>". The slash form keeps the column
// narrow while making the boundary visible at a glance.
func formatTokenOrgScope(t turso.ApiToken) string {
	switch {
	case t.Organization == "":
		return "all"
	case t.Group == "":
		return t.Organization
	default:
		return t.Organization + "/" + t.Group
	}
}

// formatTokenPermissions summarizes the scope list for the "Permissions"
// column. The platform expands presets to their underlying scopes server-side,
// so we recognize the canonical preset shapes here ("read-only" = just
// `read`, "full-access" = every known scope) for a friendlier label;
// anything else is listed verbatim.
func formatTokenPermissions(t turso.ApiToken) string {
	if len(t.Scopes) == 0 {
		return "—"
	}
	scopes := append([]string(nil), t.Scopes...)
	sort.Strings(scopes)
	if len(scopes) == 1 && scopes[0] == string(turso.ScopeRead) {
		return "read-only"
	}
	if isFullAccessScopeSet(scopes) {
		return "full-access"
	}
	return strings.Join(scopes, ", ")
}

func isFullAccessScopeSet(sortedScopes []string) bool {
	if len(sortedScopes) != len(turso.AllScopes) {
		return false
	}
	reference := make([]string, 0, len(turso.AllScopes))
	for _, s := range turso.AllScopes {
		reference = append(reference, string(s))
	}
	sort.Strings(reference)
	for i, s := range sortedScopes {
		if s != reference[i] {
			return false
		}
	}
	return true
}
