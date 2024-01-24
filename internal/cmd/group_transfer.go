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
	groupCmd.AddCommand(groupTransferCmd)
	flags.AddYes(groupTransferCmd, "Confirms the transfer")
}

var groupTransferCmd = &cobra.Command{
	Use:               "transfer <group-name> <organization-name>",
	Short:             "Transfers the group to the specified organization",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: groupTransferArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		group := args[0]
		org := args[1]

		if _, err := getGroup(client, group); err != nil {
			return err
		}

		if yesFlag {
			return transferGroup(client, group, org)
		}

		fmt.Printf("Once group %s is transfered, all its locations and databases will belong to organization %s.\n\n", internal.Emph(group), internal.Emph(org))
		fmt.Println(internal.Warn("Warning:"))
		fmt.Println("\tHostnames from all databases will be updated to use the new organization name.")
		fmt.Println("\tOld hostnames will keep working until the database name is reused on the current organization.")
		fmt.Printf("\tMake sure to update connection URL used by your applications as soon as the transfer is done to avoid disruptions.\n\n")

		ok, err := promptConfirmation("Are you sure you want to do this?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Group transfer skipped by the user.")
			return nil
		}

		return transferGroup(client, group, org)
	},
}

func transferGroup(client *turso.Client, group, organization string) error {
	msg := fmt.Sprintf("Transfering group %s to organization %s", internal.Emph(group), internal.Emph(organization))
	s := prompt.Spinner(msg)
	defer s.Stop()

	if err := client.Groups.Transfer(group, organization); err != nil {
		return fmt.Errorf("error transfering group: %w", err)
	}

	s.Stop()
	fmt.Printf("✔  Success! Group %s was transfered successfully to organization %s\n", internal.Emph(group), internal.Emph(organization))
	return nil
}

func groupTransferArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return groupArgs(cmd, args, toComplete)
	case 1:
		return organizationArgs(cmd, args, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
}

func organizationArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}
	orgs, _ := listOrganizations(client)
	slugs := make([]string, 0, len(orgs))
	for _, org := range orgs {
		slugs = append(slugs, org.Slug)
	}
	return slugs, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
