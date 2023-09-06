package cmd

import (
	"fmt"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var groupLocationsCmd = &cobra.Command{
	Use:   "locations",
	Short: "Manage your database group locations",
}

func init() {
	groupCmd.AddCommand(groupLocationsCmd)
	groupLocationsCmd.AddCommand(groupLocationsListCmd)
	addPersistentGroupFlag(groupLocationsCmd, "Use the specified group as target for the operation")
	groupLocationsCmd.AddCommand(groupLocationAddCmd)
	groupLocationsCmd.AddCommand(groupsLocationsRmCmd)
}

var groupLocationsListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List database group locations",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		if groupFlag == "" {
			return fmt.Errorf("the group flag is required")
		}

		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		groups, err := client.Groups.Get(groupFlag)
		if err != nil {
			return err
		}

		fmt.Println(formatLocations(groups.Locations, groups.Primary))
		return nil
	},
}

var groupLocationAddCmd = &cobra.Command{
	Use:               "add [...locations]",
	Short:             "Add locations to a database group",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		if groupFlag == "" {
			return fmt.Errorf("the group flag is required")
		}

		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		for _, location := range args {
			if !isValidLocation(client, location) {
				return fmt.Errorf("location '%s' is not a valid one", location)
			}
		}

		start := time.Now()
		spinner := prompt.Spinner("")
		defer spinner.Stop()

		for _, location := range args {
			description := fmt.Sprintf("Replicating group %s to %s...", internal.Emph(groupFlag), internal.Emph(location))
			spinner.Text(description)

			if err := client.Groups.AddLocation(groupFlag, location); err != nil {
				return fmt.Errorf("failed to replicate group %s to %s: %w", groupFlag, location, err)
			}
		}

		spinner.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Group %s replicated to %d locations in %d seconds.\n", internal.Emph(groupFlag), len(args), int(elapsed.Seconds()))
		return nil
	},
}

var groupsLocationsRmCmd = &cobra.Command{
	Use:               "remove [...locations]",
	Short:             "Remove locations from a database group",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		if groupFlag == "" {
			return fmt.Errorf("the group flag is required")
		}

		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		group, err := client.Groups.Get(groupFlag)
		if err != nil {
			return err
		}

		for _, location := range args {
			if !isValidLocation(client, location) {
				return fmt.Errorf("location '%s' is not a valid one", location)
			}
			if group.Primary == location {
				return fmt.Errorf("cannot remove primary location '%s' from group '%s'", location, groupFlag)
			}
		}

		start := time.Now()
		spinner := prompt.Spinner("")
		defer spinner.Stop()

		for _, location := range args {
			description := fmt.Sprintf("Removing group %s from %s...", internal.Emph(groupFlag), internal.Emph(location))
			spinner.Text(description)

			if err := client.Groups.RemoveLocation(groupFlag, location); err != nil {
				return fmt.Errorf("failed to remove group %s from %s: %w", groupFlag, location, err)
			}
		}

		spinner.Stop()
		elapsed := time.Since(start)
		fmt.Printf("Group %s removed from %d locations in %d seconds.\n", internal.Emph(groupFlag), len(args), int(elapsed.Seconds()))
		return nil
	},
}
