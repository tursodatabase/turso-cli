package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

var groupAwsMigrationCmd = &cobra.Command{
	Use:   "migration",
	Short: "Manage AWS migration of the group",
}

func init() {
	groupCmd.AddCommand(groupAwsMigrationCmd)
	groupAwsMigrationCmd.AddCommand(groupAwsMigrationInfoCmd)
	groupAwsMigrationCmd.AddCommand(groupAwsMigrationStartCmd)
}

var groupAwsMigrationInfoCmd = &cobra.Command{
	Use:               "info <group-name>",
	Short:             "Migration status for the group",
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

		_, err = client.Groups.Get(group)
		if err != nil {
			return err
		}

		info, err := client.Groups.GetAwsMigrationInfo(group)
		if err != nil {
			return err
		}

		if info.Status == "pending" {
			fmt.Printf("Migration is %v\n%v", internal.Emph("in progress"), info.Comment)
		} else if info.Status == "finished" {
			fmt.Printf("Migration is %v", internal.Emph("finished"))
		} else if info.Status == "none" {
			fmt.Printf("Migration is %v\n\n%v", internal.Emph("not started"), info.Comment)
		}

		return nil
	},
}

var groupAwsMigrationStartCmd = &cobra.Command{
	Use:               "start <group-name>",
	Short:             "Start migration process of the group",
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

		_, err = client.Groups.Get(group)
		if err != nil {
			return err
		}

		info, err := client.Groups.GetAwsMigrationInfo(group)
		if err != nil {
			return err
		}

		if info.Status == "pending" {
			fmt.Printf("Migration is %v\n%v", internal.Emph("in progress"), info.Comment)
			return nil
		} else if info.Status == "finished" {
			fmt.Printf("Migration is %v", internal.Emph("finished"))
			return nil
		}

		fmt.Printf("%v\n\n", info.Comment)

		ok, err := promptConfirmation(fmt.Sprintf("Are you sure you want to migrate group %s from Fly to AWS?", internal.Emph(group)))
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Group migration cancelled by the user.")
			return nil
		}

		spinner := prompt.Spinner(fmt.Sprintf("Migration of group %v is in progress", group))
		defer spinner.Stop()

		err = client.Groups.StartAwsMigration(group)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				spinner.Stop()
				fmt.Printf("Group %v migration still on-going\nCheck status few minutes later and contact support@turso.tech in case of any issues", internal.Emph(group))
				return nil
			case <-time.NewTimer(5 * time.Second).C:
				info, err := client.Groups.GetAwsMigrationInfo(group)
				if err != nil {
					return err
				}
				if info.Status == "finished" {
					spinner.Stop()
					fmt.Printf("Group %v was successfully migrated from Fly to AWS", internal.Emph(group))
					return nil
				} else {
					spinner.Text(info.Comment)
				}
			}
		}
	},
}
