package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

func init() {
	apiTokensCmd.AddCommand(createApiTokensCmd)
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
		description := fmt.Sprintf("Creating api token %s", internal.Emph(name))
		bar := prompt.Spinner(description)
		defer bar.Stop()

		data, err := client.ApiTokens.Create(name)
		if err != nil {
			return err
		}

		bar.Stop()
		fmt.Println(data.Value)
		return nil
	},
}
