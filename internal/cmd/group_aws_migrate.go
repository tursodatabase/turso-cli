package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
)

var groupAwsMigrationCmd = &cobra.Command{
	Use:   "aws-migration",
	Short: "Manage AWS migration of the group",
}

func init() {
	groupCmd.AddCommand(groupAwsMigrationCmd)
	groupAwsMigrationCmd.AddCommand(groupAwsMigrationInfoCmd)
	groupAwsMigrationCmd.AddCommand(groupAwsMigrationStartCmd)
	groupAwsMigrationCmd.AddCommand(groupAwsMigrationAbortCmd)
}

var groupAwsMigrationInfoCmd = &cobra.Command{
	Use:               "info <group-name>",
	Short:             "Migration status for the group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		group := args[0]
		if group == "" {
			return errors.New("the first argument must contain a group name")
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
			fmt.Printf("AWS migration is %v\n%v\n", internal.Emph("in progress"), info.Comment)
		} else if info.Status == "finished" {
			fmt.Printf("AWS migration is %v\n", internal.Emph("finished"))
		} else if info.Status == "aborted" {
			fmt.Printf("AWS migration was %v\n", internal.Emph("aborted"))
		} else if info.Status == "none" {
			fmt.Printf("AWS migration is %v\n", internal.Emph("not started"))
		}

		return nil
	},
}

var groupAwsMigrationStartCmd = &cobra.Command{
	Use:               "start <group-name>",
	Short:             "Start AWS migration process of the group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		group := args[0]
		if group == "" {
			return errors.New("the first argument must contain a group name")
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
			fmt.Printf("AWS migration is %v\n%v\n", internal.Emph("in progress"), info.Comment)
			return nil
		} else if info.Status == "finished" {
			fmt.Printf("AWS migration is %v\n", internal.Emph("finished"))
			return nil
		} else if info.Status == "aborted" {
			fmt.Printf("AWS migration was %v\nPlease, contact with support@turso.tech for further assistance with group migration\n", internal.Emph("aborted"))
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

		spinner := prompt.Spinner(fmt.Sprintf("AWS migration of group %v is in progress", group))
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
				fmt.Printf("AWS migration for group %v still on-going.\n\n"+
					"You can check status of the migration with: `turso group migration info <group-name>`.\n"+
					"If migrations for certain databases haven't started, you can abort the group migration with: `turso group migration abort <group-name>`.\n\n"+
					"Contact support@turso.tech in case of any issues\n", internal.Emph(group))
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

var groupAwsMigrationAbortCmd = &cobra.Command{
	Use:               "abort <group-name>",
	Short:             "Abort AWS migration process of the group",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		group := args[0]
		if group == "" {
			return errors.New("the first argument must contain a group name")
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

		if info.Status == "none" {
			fmt.Printf("AWS migration is %v", internal.Emph("not started"))
			return nil
		} else if info.Status == "aborted" {
			fmt.Printf("AWS migration was already %v", internal.Emph("aborted"))
			return nil
		} else if info.Status == "finished" {
			fmt.Printf("AWS migration was already %v", internal.Emph("finished"))
			return nil
		}

		ok, err := promptConfirmation(fmt.Sprintf("Are you sure you want to abort group migration %s from Fly to AWS?", internal.Emph(group)))
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}

		if !ok {
			fmt.Println("Group migration abort cancelled by the user.")
			return nil
		}

		spinner := prompt.Spinner(fmt.Sprintf("AWS migration of group %v aborted", group))
		defer spinner.Stop()

		return client.Groups.AbortAwsMigration(group)
	},
}
