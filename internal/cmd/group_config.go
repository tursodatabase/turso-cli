package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	groupCmd.AddCommand(groupConfigCmd)
	groupConfigCmd.AddCommand(groupDeleteProtectionCmd)
	groupDeleteProtectionCmd.AddCommand(groupEnableDeleteProtectionCmd)
	groupDeleteProtectionCmd.AddCommand(groupDisableDeleteProtectionCmd)
	groupDeleteProtectionCmd.AddCommand(groupShowDeleteProtectionCmd)
}

var groupConfigCmd = &cobra.Command{
	Use:               "config",
	Short:             "Manage group config",
	ValidArgsFunction: noSpaceArg,
}

var groupDeleteProtectionCmd = &cobra.Command{
	Use:               "delete-protection",
	Short:             "Manage delete-protection config of a group",
	ValidArgsFunction: noSpaceArg,
}

var groupEnableDeleteProtectionCmd = &cobra.Command{
	Use:               "enable <group-name>",
	Short:             "Disables delete protection for this group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateGroupDeleteProtection(args[0], true)
	},
}

var groupDisableDeleteProtectionCmd = &cobra.Command{
	Use:               "disable <group-name>",
	Short:             "Disables delete protection for this group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return updateGroupDeleteProtection(args[0], false)
	},
}

var groupShowDeleteProtectionCmd = &cobra.Command{
	Use:               "show <group-name>",
	Short:             "Shows the delete protection status of a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		name := args[0]

		group, err := getGroup(client, name)
		if err != nil {
			return err
		}
		config, err := client.Groups.GetConfig(group.Name)
		if err != nil {
			return err
		}
		fmt.Print(groupDeleteProtectionMessage(config.IsDeleteProtected()))
		return err
	},
}

func updateGroupDeleteProtection(name string, deleteProtection bool) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}
	group, err := getGroup(client, name)
	if err != nil {
		return err
	}
	return client.Groups.UpdateConfig(group.Name, turso.GroupConfig{DeleteProtection: &deleteProtection})
}

func groupDeleteProtectionMessage(status bool) string {
	msg := "off"
	if status {
		msg = "on"
	}
	return fmt.Sprintf("Delete Protection %s\n", internal.Emph(msg))
}
