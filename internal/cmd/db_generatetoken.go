package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var (
	groupTokenFlag bool
)

func init() {
	dbTokensCmd.AddCommand(dbGenerateTokenCmd)

	flags.AddExpiration(dbGenerateTokenCmd)
	flags.AddReadOnly(dbGenerateTokenCmd)
	flags.AddAttachClaims(dbGenerateTokenCmd)
	dbGenerateTokenCmd.Flags().BoolVar(&groupTokenFlag, "group", false, "create a token that is valid for all databases in the group")
}

var dbGenerateTokenCmd = &cobra.Command{
	Use:               "create <database-name>",
	Short:             "Creates a bearer token to authenticate requests to the database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
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

		var claim *turso.PermissionsClaim
		if len(flags.AttachClaims()) > 0 {
			err := validateDBNames(client, flags.AttachClaims())
			if err != nil {
				return err
			}
			claim = &turso.PermissionsClaim{
				ReadAttach: turso.Entities{DBNames: flags.AttachClaims()},
			}
		}
		token, err := getToken(client, database, expiration, flags.ReadOnly(), groupTokenFlag, claim)
		if err != nil {
			return fmt.Errorf("your database does not support token generation")
		}
		fmt.Println(token)
		return nil
	},
}

func getToken(client *turso.Client, database turso.Database, expiration string, readOnly, group bool, claim *turso.PermissionsClaim) (string, error) {
	if !group {
		return client.Databases.Token(database.Name, expiration, readOnly, claim)
	}
	if group && database.Group == "" {
		return "", fmt.Errorf("--group flag can only be set with group databases")
	}
	return client.Groups.Token(database.Group, expiration, readOnly, claim)
}
