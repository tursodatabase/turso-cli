package cmd

import (
	"fmt"
	"syscall"

	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func init() {
	dbCmd.AddCommand(changePasswordCmd)

	changePasswordCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Value of new password to be set on database")
	changePasswordCmd.RegisterFlagCompletionFunc("password", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}

var changePasswordCmd = &cobra.Command{
	Use:               "passwd database_name",
	Short:             "Change password to all instances of the database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClient()
		if err != nil {
			return err
		}
		db, err := getDatabase(client, args[0])
		if err != nil {
			return err
		}

		config, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		var newPassword string
		if len(passwordFlag) > 0 {
			newPassword = passwordFlag
		} else {
			fmt.Print("Enter new password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("unable to read the new password: %s", err)
			}
			newPassword = string(bytePassword)
		}

		bar := prompt.Spinner("Changing password...")
		defer bar.Stop()
		client, err = createTursoClient()
		if err != nil {
			return err
		}
		err = client.Databases.ChangePassword(args[0], newPassword)
		bar.Stop()
		if err != nil {
			return err
		}
		err = config.SetDatabasePassword(db.ID, newPassword)
		if err != nil {
			return fmt.Errorf("password changed but failed to persist in locally. Please retry. Error: %s", err)
		}
		fmt.Println("Password changed succesfully!")
		return nil
	},
}
