package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func RetryOnSleepingGroup(client *turso.Client, group string, mainSpinnerText string, action func() (any, error)) (any, error) {
	mainSpinner := prompt.Spinner(mainSpinnerText)
	result, actionErr := action()
	if actionErr == nil {
		return result, nil
	}

	errMsg := actionErr.Error()
	if strings.Contains(errMsg, "group_sleeping") ||
		(strings.Contains(errMsg, "cannot create database on group") && strings.Contains(errMsg, "because it is archived")) {

		mainSpinner.Stop()
		fmt.Printf("Error: %s\n\n", errMsg)
		time.Sleep(time.Second)

		promptMsg := fmt.Sprintf("The group %s is currently archived. Do you want to unarchive it now??", internal.Emph(group))
		confirmed, err := promptConfirmation(promptMsg)
		if err != nil {
			return nil, fmt.Errorf("could not get prompt confirmed by user: %w", err)
		}
		if !confirmed {
			return nil, fmt.Errorf("cannot perform action on an archived group")
		}

		err = unarchiveGroup(client, group)
		if err != nil {
			return nil, fmt.Errorf("failed to wake up group: %w", err)
		}

		fmt.Printf("Retrying...\n")
		time.Sleep(time.Second)

		mainSpinner = prompt.Spinner(mainSpinnerText)

		return action()
	}

	mainSpinner.Stop()
	return nil, actionErr
}
