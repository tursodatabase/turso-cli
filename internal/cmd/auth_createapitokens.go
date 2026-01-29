package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

var mintOrgFlag string

func init() {
	apiTokensCmd.AddCommand(createApiTokensCmd)
	createApiTokensCmd.Flags().StringVar(&mintOrgFlag, "org", "", "Organization to restrict the token to")
}

var createApiTokensCmd = &cobra.Command{
	Use:   "mint <api-token-name>",
	Short: "Mint an API token.",
	Long: "" +
		"API tokens are revocable non-expiring tokens that authenticate holders as the user who minted them.\n" +
		"They can be used to implement automations with the " + internal.Emph("turso") + " CLI or the platform API.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		name := strings.TrimSpace(args[0])

		// Validate organization if provided
		if mintOrgFlag != "" {
			orgs, err := client.Organizations.List()
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			found := false
			for _, org := range orgs {
				if org.Slug == mintOrgFlag {
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("organization %s not found", internal.Emph(mintOrgFlag))
			}
		}

		data, err := client.ApiTokens.CreateWithOrg(name, mintOrgFlag)
		if err != nil {
			return err
		}

		fmt.Println(data.Value)
		return nil
	},
}
