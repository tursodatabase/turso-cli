package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	groupCmd.AddCommand(groupProtectionCmd)
	groupProtectionCmd.AddCommand(groupProtectionEnableCmd)
	groupProtectionCmd.AddCommand(groupProtectionDisableCmd)
	groupProtectionCmd.AddCommand(groupProtectionShowCmd)
}

var groupProtectionCmd = &cobra.Command{
	Use:               "protection",
	Short:             "Manage delete protection of a group",
	ValidArgsFunction: noSpaceArg,
}

var groupProtectionEnableCmd = &cobra.Command{
	Use:               "enable <group-name>",
	Short:             "Enable delete protection for a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return setGroupProtection(args[0], true)
	},
}

var groupProtectionDisableCmd = &cobra.Command{
	Use:               "disable <group-name>",
	Short:             "Disable delete protection for a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return setGroupProtection(args[0], false)
	},
}

var groupProtectionShowCmd = &cobra.Command{
	Use:               "show <group-name>",
	Short:             "Shows the delete protection status of a group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: groupArgs,
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
		fmt.Print(protectionMessage(config.IsDeleteProtected()))
		return nil
	},
}

func setGroupProtection(name string, protect bool) error {
	client, err := authedTursoClient()
	if err != nil {
		return err
	}

	_, err = getGroup(client, name)
	if err != nil {
		return err
	}

	deleteProtection := protect
	config := turso.GroupConfig{
		DeleteProtection: &deleteProtection,
	}

	err = client.Groups.UpdateConfig(name, config)
	if err != nil {
		action := "enable"
		if !protect {
			action = "disable"
		}
		return fmt.Errorf("failed to %s delete protection for group %s: %w", action, name, err)
	}

	fmt.Print(protectionMessage(protect))
	return nil
}
