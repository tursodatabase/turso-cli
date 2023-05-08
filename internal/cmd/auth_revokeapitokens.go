package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/spf13/cobra"
)

func init() {
	apiTokensCmd.AddCommand(revokeApiTokensCmd)
}

var revokeApiTokensCmd = &cobra.Command{
	Use:   "revoke api_token_name",
	Short: "Revoke an API token.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		ok, err := promptConfirmation("Are you sure you want to revoke this token?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Revocation skipped by the user.")
			return nil
		}

		tokenName := args[0]

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
