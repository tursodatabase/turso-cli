package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var timestampS string
var fromNow time.Duration

func addTimestampFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&timestampS, "timestamp", "", "Set timestamp target (RFC3339 format)")
	cmd.Flags().DurationVar(&fromNow, "from-now", time.Duration(0), "Set timestamp target relative from now")
}

func getTimestamp(cmd *cobra.Command) (*time.Time, error) {
	isTimestampSet := cmd.Flag("timestamp").Changed
	isFromNowSet := cmd.Flag("from-now").Changed

	if isTimestampSet && isFromNowSet {
		return nil, fmt.Errorf("cannot set both timestamp and from-now flags")
	}

	if !isTimestampSet && !isFromNowSet {
		return nil, nil
	}

	if isTimestampSet {
		timestamp, err := time.Parse(time.RFC3339, timestampS)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format: %w", err)
		}
		return &timestamp, nil
	}

	timestamp := time.Now().Add(-fromNow)
	return &timestamp, nil
}
