package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

var expFlag expirationFlag
var readOnly bool

func init() {
	dbTokensCmd.AddCommand(dbGenerateTokenCmd)

	usage := fmt.Sprintf("Token expiration. Possible values are %s (default) or expiration time in days (e.g. %s).", internal.Emph("never"), internal.Emph("7d"))
	dbGenerateTokenCmd.Flags().VarP(&expFlag, "expiration", "e", usage)
	dbGenerateTokenCmd.RegisterFlagCompletionFunc("expiration", expirationFlagCompletion)

	dbGenerateTokenCmd.Flags().BoolVarP(&readOnly, "read-only", "r", false, "Token with read-only access")
}

var dbGenerateTokenCmd = &cobra.Command{
	Use:               "create database_name",
	Short:             "Creates a bearer token to authenticate requests to the database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]

		if _, err := getDatabase(client, name); err != nil {
			return err
		}

		token, err := client.Databases.Token(name, expFlag.String(), readOnly)
		if err != nil {
			return fmt.Errorf("your database does not support token generation")
		}
		fmt.Println(token)
		return nil
	},
}
