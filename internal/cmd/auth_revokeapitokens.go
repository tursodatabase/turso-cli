package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

func init() {
	apiTokensCmd.AddCommand(revokeApiTokensCmd)
}

var revokeApiTokensCmd = &cobra.Command{
	Use:   "revoke <api-token-name>",
	Short: "Revoke an API token.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		tokenName := args[0]

		apiTokens, err := client.ApiTokens.List()
		if err != nil {
			return err
		}

		found := false
		for _, apiToken := range apiTokens {
			if apiToken.Name == tokenName {
				found = true
				break
			}
		}

		if !found {
			fmt.Println("API token not found, revocation skipped.")
			return nil
		}

		ok, err := promptConfirmation("Are you sure you want to revoke this token?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Revocation skipped by the user.")
			return nil
		}

		s := prompt.Spinner(fmt.Sprintf("Revoking API token %s... ", internal.Emph(tokenName)))
		defer s.Stop()

		if err := client.ApiTokens.Revoke(tokenName); err != nil {
			return err
		}
		s.Stop()
		fmt.Printf("API token %s successfully revoked.\n", internal.Emph(tokenName))

		return nil
	},
}
