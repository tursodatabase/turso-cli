package cmd

import (
	"fmt"
	"strings"

	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func ensureGroupAwake(client *turso.Client, groupName string) (bool, error) {
	group, err := getGroup(client, groupName)
	if err != nil {
		return false, fmt.Errorf("failed to get group info: %w", err)
	}

	groupStatus := aggregateGroupStatus(group)
	if groupStatus != "Archived 💤" {
		return true, nil
	}

	fmt.Printf("The group %s is currently archived. Do you want to wake it up? [Y/n]: ", internal.Emph(groupName))
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" || response == "y" || response == "yes" {
		err = unarchiveGroup(client, groupName)
		if err != nil {
			return false, fmt.Errorf("failed to wake up group: %w", err)
		}
		fmt.Printf("Group %s has been woken up.\n", internal.Emph(groupName))
		return true, nil
	}

	return false, nil
}
