package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbAuthCmd)
}

var dbAuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage database authentication",
}

var expFlag expirationFlag

func init() {
	dbAuthCmd.AddCommand(dbGenerateTokenCmd)
	dbAuthCmd.AddCommand(dbAuthRotateCmd)

	usage := "Token expiration. Possible values are 'default' or 'none'."
	dbGenerateTokenCmd.Flags().VarP(&expFlag, "expiration", "e", usage)
	dbGenerateTokenCmd.RegisterFlagCompletionFunc("expiration", expirationFlagCompletion)

	dbAuthRotateCmd.Flags().BoolVarP(&yesFlag, "yes", "y", false, "Confirms the rotation database credentials.")
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

var dbAuthRotateCmd = &cobra.Command{
	Use:               "invalidate-tokens database_name",
	Short:             "Rotates the keys used to create and verify database tokens making existing tokens invalid",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		name := args[0]

		if _, err := getDatabase(client, name); err != nil {
			return err
		}

		if yesFlag {
			return rotate(client, name)
		}

		fmt.Printf("To rotate %s database credentials, all its replicas must be restarted.\n", turso.Emph(name))
		fmt.Printf("All your acitve connections to the DB will be dropped and there will be a short downtime.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Credentials rotation skipped by the user.")
			return nil
		}

		return rotate(client, name)
	},
}

func rotate(turso *turso.Client, name string) error {
	s := startLoadingBar("Rotating database keys... ")

	if err := turso.Databases.Rotate(name); err != nil {
		s.Stop()
		return fmt.Errorf("your database does not support tokens")
	}

	s.Stop()
	fmt.Println("✔  Success! Keys rotated successfully")
	return nil
}
