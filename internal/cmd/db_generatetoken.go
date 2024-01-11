package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var (
	readOnlyFlag   bool
	groupTokenFlag bool
)

func init() {
	dbTokensCmd.AddCommand(dbGenerateTokenCmd)

	flags.AddExpiration(dbGenerateTokenCmd)
	dbGenerateTokenCmd.Flags().BoolVar(&groupTokenFlag, "group", false, "create a token that is valid for all databases in the group")
	dbGenerateTokenCmd.Flags().BoolVarP(&readOnlyFlag, "read-only", "r", false, "Token with read-only access")
}

var dbGenerateTokenCmd = &cobra.Command{
	Use:               "create <database-name>",
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

		database, err := getDatabase(client, name, true)
		if err != nil {
			return err
		}

		expiration, err := flags.Expiration()
		if err != nil {
			return err
		}

		token, err := getToken(client, database, expiration, readOnlyFlag, groupTokenFlag)
		if err != nil {
			return fmt.Errorf("your database does not support token generation")
		}
		fmt.Println(token)
		return nil
	},
}

func getToken(client *turso.Client, database turso.Database, expiration string, readOnly, group bool) (string, error) {
	if !group {
		return client.Databases.Token(database.Name, expiration, readOnly)
	}
	if group && database.Group == "" {
		return "", fmt.Errorf("--group flag can only be set with group databases")
	}
	return client.Groups.Token(database.Group, expiration, readOnly)
}
