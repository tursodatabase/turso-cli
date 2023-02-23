package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
)

func changePasswordShellArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return getDatabaseNames(createTursoClient()), cobra.ShellCompDirectiveNoFileComp
	}
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

var changePasswordCmd = &cobra.Command{
	Use:               "change-password database_name new_password",
	Short:             "Change password to all instances of the database",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: changePasswordShellArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client := createTursoClient()
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		if db.Type != "logical" {
			return fmt.Errorf("only new databases, of type 'logical', support the change password operation")
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		bar := startLoadingBar("Changing password...")
		defer bar.Stop()
		err = createTursoClient().Databases.ChangePassword(args[0], args[1])
		bar.Stop()
		if err != nil {
			return err
		}
		err = config.SetDatabasePassword(db.ID, args[1])
		if err != nil {
			return fmt.Errorf("password changed but failed to persist in locally. Please retry. Error: %s", err)
		}
		fmt.Println("Password changed succesfully!")
		return nil
	},
}
