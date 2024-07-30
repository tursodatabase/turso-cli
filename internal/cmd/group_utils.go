package cmd

import (
	"fmt"

	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func ensureGroupAwake(client *turso.Client, groupName string) (bool, error) {
	group, err := getGroup(client, groupName)
	if err != nil {
		return false, fmt.Errorf("failed to get group info: %w", err)
	}

	groupStatus := aggregateGroupStatus(group)
	if groupStatus != "Archived" {
		return true, nil
	}

	prompt := fmt.Sprintf("The group %s is currently archived. Do you want to wake it up?", internal.Emph(groupName))
	confirmed, err := promptConfirmation(prompt)
	if err != nil {
		return false, fmt.Errorf("could not get prompt confirmed by user: %w", err)
	}

	if confirmed {
		err = unarchiveGroup(client, groupName)
		if err != nil {
			return false, fmt.Errorf("failed to wake up group: %w", err)
		}
		return true, nil
	}

	return false, nil
}
