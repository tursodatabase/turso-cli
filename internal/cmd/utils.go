package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal/settings"
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

func getDatabaseUrl(settings *settings.Settings, db turso.Database) string {
	dbSettings := settings.GetDatabaseSettings(db.ID)
	if dbSettings == nil {
		// Backwards compatibility with old settings files.
		dbSettings = settings.GetDatabaseSettings(db.Name)
	}

	url := "<n/a>"
	if dbSettings != nil {
		url = dbSettings.GetURL()
	}
	return url
}
