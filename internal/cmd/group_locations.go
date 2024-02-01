package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"golang.org/x/exp/maps"
)

var groupLocationsCmd = &cobra.Command{
	Use:   "locations",
	Short: "Manage your database group locations",
}

func init() {
	groupCmd.AddCommand(groupLocationsCmd)
	groupLocationsCmd.AddCommand(groupLocationsListCmd)
	groupLocationsCmd.AddCommand(groupLocationAddCmd)
	addWaitFlag(groupLocationAddCmd, "Wait for group location to be ready")
	groupLocationsCmd.AddCommand(groupsLocationsRmCmd)
}

var groupLocationsListCmd = &cobra.Command{
	Use:               "list <group-name>",
	Short:             "List database group locations",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		group := args[0]
		if group == "" {
			return fmt.Errorf("the first argument must contain a group name")
		}

		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		groups, err := client.Groups.Get(group)
		if err != nil {
			return err
		}

		fmt.Println(formatLocations(groups.Locations, groups.Primary))
		return nil
	},
}

var groupLocationAddCmd = &cobra.Command{
	Use:               "add <group-name> <...location-code>",
	Short:             "Add locations to a database group",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: locationsAddArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName := args[0]
		if groupName == "" {
			return fmt.Errorf("the first argument must contain a group name")
		}

		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		group, err := client.Groups.Get(groupName)
		if err != nil {
			return err
		}
		alreadyExistingLocations := map[string]bool{}
		for _, location := range group.Locations {
			alreadyExistingLocations[location] = true
		}

		locations := args[1:]
		for _, location := range locations {
			if !isValidLocation(client, location) {
				return fmt.Errorf("location '%s' is not a valid one", location)
			}
			if alreadyExistingLocations[location] {
				return fmt.Errorf("location '%s' is already part of group '%s'", location, groupName)
			}
		}

		start := time.Now()
		spinner := prompt.Spinner("")
		defer spinner.Stop()

		invalidateGroupsCache(client.Org)
		for _, location := range locations {
			description := fmt.Sprintf("Replicating group %s to %s...", internal.Emph(groupName), internal.Emph(location))
			spinner.Text(description)

			if err := client.Groups.AddLocation(groupName, location); err != nil {
				return fmt.Errorf("failed to replicate group %s to %s: %w", groupName, location, err)
			}

			if err := handleGroupWaitFlag(client, groupName, location); err != nil {
				return fmt.Errorf("failed to wait for group %s to be ready on location %s: %w", groupName, location, err)
			}
		}

		spinner.Stop()
		elapsed := time.Since(start)

		if len(locations) == 1 {
			fmt.Printf("Group %s replicated to %s in %d seconds.\n", internal.Emph(groupName), internal.Emph(locations[0]), int(elapsed.Seconds()))
			return nil
		}

		fmt.Printf("Group %s replicated to %d locations in %d seconds.\n", internal.Emph(groupName), len(locations), int(elapsed.Seconds()))
		return nil
	},
}

var groupsLocationsRmCmd = &cobra.Command{
	Use:               "remove <group-name> <...location-code>",
	Short:             "Remove locations from a database group",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: locationsRmArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		groupName := args[0]
		if groupName == "" {
			return fmt.Errorf("the group flag is required")
		}

		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		group, err := client.Groups.Get(groupName)
		if err != nil {
			return err
		}

		locations := args[1:]
		for _, location := range locations {
			if !isValidLocation(client, location) {
				return fmt.Errorf("location '%s' is not a valid one", location)
			}
			if group.Primary == location {
				return fmt.Errorf("cannot remove primary location '%s' from group '%s'", location, groupName)
			}
		}

		start := time.Now()
		spinner := prompt.Spinner("")
		defer spinner.Stop()

		invalidateGroupsCache(client.Org)
		for _, location := range locations {
			description := fmt.Sprintf("Removing group %s from %s...", internal.Emph(groupName), internal.Emph(location))
			spinner.Text(description)

			if err := client.Groups.RemoveLocation(groupName, location); err != nil {
				return fmt.Errorf("failed to remove group %s from %s: %w", groupName, location, err)
			}
		}

		spinner.Stop()
		elapsed := time.Since(start)

		if len(locations) == 1 {
			fmt.Printf("Group %s removed from %s in %d seconds.\n", internal.Emph(groupName), internal.Emph(locations[0]), int(elapsed.Seconds()))
			return nil
		}

		fmt.Printf("Group %s removed from %d locations in %d seconds.\n", internal.Emph(groupName), len(locations), int(elapsed.Seconds()))
		return nil
	},
}

func locationsAddArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return groupArgs(cmd, args, toComplete)
	}

	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	locations, _ := locations(client)

	used := args[1:]
	for _, location := range used {
		delete(locations, location)
	}

	group, _ := getGroup(client, args[0])
	for _, location := range group.Locations {
		delete(locations, location)
	}

	return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func locationsRmArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return groupArgs(cmd, args, toComplete)
	}

	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	group, _ := getGroup(client, args[0])
	locations := make(map[string]bool, len(group.Locations))
	for _, location := range group.Locations {
		locations[location] = true
	}

	used := args[1:]
	for _, location := range used {
		delete(locations, location)
	}

	return maps.Keys(locations), cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func groupArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	client, err := authedTursoClient()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	groups, _ := groupNames(client)
	return groups, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
