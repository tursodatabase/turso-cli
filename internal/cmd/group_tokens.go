package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	groupCmd.AddCommand(groupTokensCmd)
}

var groupTokensCmd = &cobra.Command{
	Use:               "tokens",
	Short:             "Manage group tokens",
	ValidArgsFunction: noSpaceArg,
}

func init() {
	groupTokensCmd.AddCommand(groupTokensInvalidateCmd)
	flags.AddYes(groupTokensInvalidateCmd, "Confirms the invalidation of the credentials of the group and all its databases")
}

var groupTokensInvalidateCmd = &cobra.Command{
	Use:               "invalidate <group-name>",
	Short:             "Rotates the keys used to create and verify database tokens, invalidating all existing tokens invalid for the group.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]

		group, err := getGroup(client, name)
		if err != nil {
			return err
		}

		if flags.Yes() {
			return rotateGroup(client, group)
		}

		fmt.Printf("To invalidate tokens for group %s, tokens from all its databases will be invalidated.\n", internal.Emph(name))
		fmt.Printf("All your active connections to the databases in that group will be dropped and there will be a short downtime.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Token invalidation skipped by the user.")
			return nil
		}

		return rotateGroup(client, group)
	},
}

func rotateGroup(turso *turso.Client, group turso.Group) error {
	s := prompt.Spinner("Invalidating group tokens... ")
	defer s.Stop()

	invalidateDbTokenCache()
	settings.PersistChanges()

	if err := turso.Groups.Rotate(group.Name); err != nil {
		return err
	}

	s.Stop()
	fmt.Printf("✔  Success! Tokens invalidated successfully.\n\n")
	fmt.Printf("Run %s to get a new one.\n", internal.Emph("turso group tokens create <group-name>"))
	return nil
}

func init() {
	groupTokensCmd.AddCommand(groupCreateTokenCmd)
}

var groupCreateTokenCmd = &cobra.Command{
	Use:               "create <group-name>",
	Short:             "Creates a bearer token to authenticate to group databases",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		name := args[0]

		group, err := getGroup(client, name)
		if err != nil {
			return err
		}

		token, err := client.Groups.Token(group.Name, "", false)
		if err != nil {
			return fmt.Errorf("error creating token: %w", err)
		}

		fmt.Println(token)
		return nil
	},
}
