package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func dbNameValidator(argIndex int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		return nil
	}
}

func regionArgValidator(argIndex int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		region := args[argIndex]
		for _, v := range regionIds {
			if v == region {
				return nil
			}
		}
		return fmt.Errorf("there is no %s region available", region)
	}
}

func findInstanceFromRegion(instances []turso.Instance, region string) *turso.Instance {
	for _, instance := range instances {
		if instance.Region == region {
			return &instance
		}
	}
	return nil
}
