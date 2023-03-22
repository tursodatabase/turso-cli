package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var expFlag expirationFlag

func init() {
	dbCmd.AddCommand(dbGenerateTokenCmd)

	usage := "Token expiration. Possible values are 'default' or 'none'."
	dbGenerateTokenCmd.Flags().VarP(&expFlag, "expiration", "e", usage)
	dbGenerateTokenCmd.RegisterFlagCompletionFunc("expiration", expirationFlagCompletion)
}

var dbGenerateTokenCmd = &cobra.Command{
	Use:               "generate-token database_name",
	Short:             "Creates a bearer token to authenticate requests to the database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		name := args[0]

		if _, err := getDatabase(client, name); err != nil {
			return err
		}

		token, err := client.Databases.Token(name, expFlag.String())
		if err != nil {
			return fmt.Errorf("your database does not support token generation")
		}
		fmt.Println(token)
		return nil
	},
}
