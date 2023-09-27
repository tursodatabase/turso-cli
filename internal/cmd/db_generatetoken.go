package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

var expFlag expirationFlag
var readOnlyFlag bool
var groupTokenFlag bool

func init() {
	dbTokensCmd.AddCommand(dbGenerateTokenCmd)

	usage := fmt.Sprintf("Token expiration. Possible values are %s (default) or expiration time in days (e.g. %s).", internal.Emph("never"), internal.Emph("7d"))
	dbGenerateTokenCmd.Flags().VarP(&expFlag, "expiration", "e", usage)
	dbGenerateTokenCmd.RegisterFlagCompletionFunc("expiration", expirationFlagCompletion)
	dbGenerateTokenCmd.Flags().BoolVar(&groupTokenFlag, "group", false, "create a token that is valid for all databases in the group")
	dbGenerateTokenCmd.Flags().MarkHidden("group")

	dbGenerateTokenCmd.Flags().BoolVarP(&readOnlyFlag, "read-only", "r", false, "Token with read-only access")
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

		database, err := getDatabase(client, name, true)
		if err != nil {
			return err
		}
		expiration := expFlag.String()
		if err := validateExpiration(expiration); err != nil {
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

func validateExpiration(expiration string) error {
	if len(expiration) == 0 {
		return nil
	}
	if expiration == "none" || expiration == "default" || expiration == "never" {
		return nil
	}
	if !strings.HasSuffix(expiration, "d") {
		return nil
	}
	daysStr := strings.TrimSuffix(expiration, "d")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return err
	}
	if days < 1 {
		return fmt.Errorf("expiration must be at least 1 day")
	}
	return nil
}
