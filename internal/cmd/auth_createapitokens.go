package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	apiTokensCmd.AddCommand(createApiTokensCmd)
}

var createApiTokensCmd = &cobra.Command{
	Use:   "mint api_token_name",
	Short: "Mint an API token.",
	Long: "" +
		"API tokens are revocable non-expiring tokens that authenticate holders as the user who minted them.\n" +
		"They can be used to implement automations with the " + internal.Emph("turso") + " CLI or the platform API.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		tokenName := strings.TrimSpace(args[0])

		if !turso.IsValidName(tokenName) {
			return errors.New("invalid name: names only support lowercase letters, numbers, and hyphens")
		}

		description := fmt.Sprintf("Creating api token %s", internal.Emph(tokenName))
		bar := prompt.Spinner(description)
		defer bar.Stop()

		data, err := client.ApiTokens.Create(tokenName)
		if err != nil {
			return err
		}

		bar.Stop()
		fmt.Println(data.Token)
		return nil
	},
}
