package cmd

import (
	"fmt"
	"syscall"

	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
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
		client, err := createTursoClientFromAccessToken(true)
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
		err = client.Databases.ChangePassword(args[0], newPassword)
		bar.Stop()
		if err != nil {
			return err
		}

		fmt.Println("Password changed succesfully!")
		return nil
	},
}
