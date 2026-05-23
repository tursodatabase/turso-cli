package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var (
	mintOrgFlag        string
	mintGroupFlag      string
	mintScopeFlags     []string
	mintReadOnlyFlag   bool
	mintFullAccessFlag bool
)

func init() {
	apiTokensCmd.AddCommand(createApiTokensCmd)
	createApiTokensCmd.Flags().StringVar(&mintOrgFlag, "org", "", "Organization to restrict the token to.")
	createApiTokensCmd.Flags().StringVar(&mintGroupFlag, "group", "", "Group inside --org to restrict the token to. Implies --org and requires at least one scope.")
	createApiTokensCmd.Flags().StringArrayVar(&mintScopeFlags, "scope", nil, "Permission scope to grant to a group-scoped token. May be repeated. Allowed values: "+scopeFlagListing()+".")
	createApiTokensCmd.Flags().BoolVar(&mintReadOnlyFlag, "read-only", false, "Shorthand for --scope read.")
	createApiTokensCmd.Flags().BoolVar(&mintFullAccessFlag, "full-access", false, "Shorthand for granting every scope. Use with care; equivalent to a deployer that can create, delete, configure, mint and rotate.")
}

var createApiTokensCmd = &cobra.Command{
	Use:   "mint <api-token-name>",
	Short: "Mint an API token.",
	Long: "" +
		"API tokens are revocable non-expiring tokens that authenticate holders as the user who minted them.\n" +
		"They can be used to implement automations with the " + internal.Emph("turso") + " CLI or the platform API.\n" +
		"\n" +
		"With --group, the token is restricted to a single group inside the organization and to the\n" +
		"set of scopes you pass via --scope (or the --read-only / --full-access shorthands).",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		name := strings.TrimSpace(args[0])

		scopes, err := resolveMintScopes()
		if err != nil {
			return err
		}

		if mintGroupFlag != "" {
			if mintOrgFlag == "" {
				return fmt.Errorf("--group requires --org")
			}
			if len(scopes) == 0 {
				return fmt.Errorf("--group requires at least one scope (use --scope, --read-only, or --full-access)")
			}
		} else if len(scopes) > 0 {
			return fmt.Errorf("--scope / --read-only / --full-access are only meaningful with --group")
		}

		if mintOrgFlag != "" {
			if err := validateOrgExists(client, mintOrgFlag); err != nil {
				return err
			}
		}

		if mintGroupFlag != "" {
			if err := validateGroupExists(client, mintOrgFlag, mintGroupFlag); err != nil {
				return err
			}
		}

		data, err := client.ApiTokens.CreateScoped(name, mintOrgFlag, mintGroupFlag, scopes)
		if err != nil {
			return err
		}

		fmt.Println(data.Value)
		return nil
	},
}

// resolveMintScopes turns the --scope / --read-only / --full-access flags
// into the list of scope strings sent to the platform. Conflicting flag
// combinations (e.g. --read-only with --full-access) are rejected, and
// individual scope labels are validated client-side so typos surface
// without a round-trip.
func resolveMintScopes() ([]string, error) {
	if mintReadOnlyFlag && mintFullAccessFlag {
		return nil, fmt.Errorf("--read-only and --full-access are mutually exclusive")
	}
	if (mintReadOnlyFlag || mintFullAccessFlag) && len(mintScopeFlags) > 0 {
		return nil, fmt.Errorf("--scope cannot be combined with --read-only or --full-access; use one or the other")
	}
	if mintReadOnlyFlag {
		return []string{"read-only"}, nil
	}
	if mintFullAccessFlag {
		return []string{"full-access"}, nil
	}
	if len(mintScopeFlags) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(mintScopeFlags))
	seen := make(map[string]struct{}, len(mintScopeFlags))
	for _, raw := range mintScopeFlags {
		s := strings.TrimSpace(raw)
		if s == "" {
			continue
		}
		if !turso.IsValidScope(s) {
			return nil, fmt.Errorf("unknown scope %s. Allowed: %s", internal.Emph(s), scopeFlagListing())
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}

func validateOrgExists(client *turso.Client, slug string) error {
	orgs, err := client.Organizations.List()
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}
	for _, org := range orgs {
		if org.Slug == slug {
			return nil
		}
	}
	return fmt.Errorf("organization %s not found", internal.Emph(slug))
}

// validateGroupExists confirms that the named group lives inside the given
// organization slug. The Groups client URL-builds from client.Org, so we
// temporarily point the client at the target org for the lookup, then
// restore. (Single-command invocation, no concurrency to worry about.)
func validateGroupExists(client *turso.Client, orgSlug, groupName string) error {
	savedOrg := client.Org
	client.Org = orgSlug
	defer func() { client.Org = savedOrg }()

	if _, err := client.Groups.Get(groupName); err != nil {
		return fmt.Errorf("group %s not found in organization %s: %w", internal.Emph(groupName), internal.Emph(orgSlug), err)
	}
	return nil
}

func scopeFlagListing() string {
	parts := make([]string, 0, len(turso.AllScopes))
	for _, s := range turso.AllScopes {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, ", ")
}
