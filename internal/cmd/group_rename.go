package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	groupCmd.AddCommand(groupRenameCmd)
	flags.AddYes(groupRenameCmd, "Confirms the update")
}

var groupRenameCmd = &cobra.Command{
	Use:               "rename <old-group-name> <new-group-name>",
	Short:             "Renames the group",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: groupArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		oldName := args[0]
		newName := args[1]

		if _, err := getGroup(client, oldName); err != nil {
			return err
		}

		if yesFlag {
			return renameGroup(client, oldName, newName)
		}

		ok, err := promptConfirmation(fmt.Sprintf("Are you sure you want to rename group %s to %s?", internal.Emph(oldName), internal.Emph(newName)))
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Group rename skipped by the user.")
			return nil
		}

		return renameGroup(client, oldName, newName)
	},
}

func renameGroup(client *turso.Client, oldName, newName string) error {
	msg := fmt.Sprintf("Renaming group %s to %s", internal.Emph(oldName), internal.Emph(newName))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Groups.Rename(oldName, newName); err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("✔  Success! Group %s was renamed to %s successfully\n", internal.Emph(oldName), internal.Emph(newName))
	return nil
}
