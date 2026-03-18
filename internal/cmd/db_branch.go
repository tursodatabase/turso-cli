package cmd

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

var branchTimestampFlag string

func init() {
	dbCmd.AddCommand(dbBranchCmd)
	addGroupFlag(dbBranchCmd)
	addLocationFlag(dbBranchCmd, "Location ID. If no ID is specified, primary location of the destination group is used by default.")
	addRemoteEncryptionCipherFlag(dbBranchCmd)
	addRemoteEncryptionKeyFlag(dbBranchCmd)
	dbBranchCmd.Flags().StringVar(&branchTimestampFlag, "timestamp", "", "Set a point in time in the past to copy data from the source database. Must be in RFC3339 format like '2023-09-29T10:16:13-03:00'")
}

var dbBranchCmd = &cobra.Command{
	Use:               "branch <source-database> <target-database>",
	Short:             "Create a branch from an existing database.",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: dbBranchArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		sourceName := args[0]

		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		sourceDB, err := getDatabase(client, sourceName, true)
		if err != nil {
			return err
		}

		targetName, err := resolveBranchTargetName(args)
		if err != nil {
			return err
		}

		prevFromDB := fromDBFlag
		prevTimestamp := timestampFlag
		prevGroup := groupFlag
		defer func() {
			fromDBFlag = prevFromDB
			timestampFlag = prevTimestamp
			groupFlag = prevGroup
		}()

		fromDBFlag = sourceName
		timestampFlag = branchTimestampFlag
		if groupFlag == "" && sourceDB.Group != "" {
			groupFlag = sourceDB.Group
		}

		return CreateDatabase(targetName)
	},
}

func dbBranchArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return dbNameArg(cmd, args, toComplete)
	}
	return noFilesArg(cmd, args, toComplete)
}

func resolveBranchTargetName(args []string) (string, error) {
	targetName := args[1]

	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return "", errors.New("branch name cannot be empty")
	}
	if targetName == args[0] {
		return "", errors.New("branch name must be different from source database name")
	}
	return targetName, nil
}
