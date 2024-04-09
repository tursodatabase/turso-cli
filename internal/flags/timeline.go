package flags

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

var timelineFlag string

func AddTimeline(cmd *cobra.Command) {
	usage := fmt.Sprintf("Select the plan timeline. Possible values are %s or %s. If not selected, current plan timeline will not be changed.", internal.Emph("monthly"), internal.Emph("yearly"))
	cmd.Flags().StringVarP(&timelineFlag, "timeline", "t", "", usage)
	_ = cmd.RegisterFlagCompletionFunc("timeline", timelineFlagCompletion)
}

func Timeline() (string, error) {
	if err := validateTimeline(timelineFlag); err != nil {
		return "", err
	}
	return timelineFlag, nil
}

func validateTimeline(timeline string) error {
	switch timeline {
	case "", "monthly", "yearly":
		return nil
	default:
		return fmt.Errorf("timeline parameter must be either 'monthly' or 'yearly'")
	}
}

func timelineFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"monthly", "yearly"}, cobra.ShellCompDirectiveDefault
}
