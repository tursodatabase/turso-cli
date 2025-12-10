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
	groupCmd.AddCommand(groupUpdateCmd)
	flags.AddYes(groupUpdateCmd, "Confirms the update")
	flags.AddVersion(groupUpdateCmd, "Version to update to. Valid values: 'latest' or 'vector'")
	flags.AddExtensions(groupUpdateCmd, "Extensions to enable. Valid values: 'all' or 'none'")
}

var groupUpdateCmd = &cobra.Command{
	Use:               "update <group-name>",
	Short:             "Updates the group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		name := args[0]

		if _, err := getGroup(client, name); err != nil {
			return err
		}

		version := flags.Version()
		extensions := flags.Extensions()

		if yesFlag {
			return updateGroup(client, name, version, extensions)
		}

		fmt.Printf("To update group %s, all its locations and databases must be updated.\n", internal.Emph(name))
		fmt.Printf("All your active connections to that group will be dropped and there will be a short downtime.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Group update skipped by the user.")
			return nil
		}

		return updateGroup(client, name, version, extensions)
	},
}

func updateGroup(client *turso.Client, name, version, extensions string) error {
	msg := fmt.Sprintf("Updating group %s", internal.Emph(name))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Groups.Update(name, version, extensions); err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("✔  Success! Group %s was updated successfully\n", internal.Emph(name))
	return nil
}

func groupArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	groups, _ := groupNames(client)
	return groups, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
