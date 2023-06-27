package cmd

import (
	"github.com/spf13/cobra"
)

type PlanInfo struct {
	maxRowsRead    uint64
	maxRowsWritten uint64
	maxDatabases   uint64
	maxStorage     uint64
	maxLocations   uint64
}

type PlanType string

const (
	ENTERPRISE PlanType = "enterprise"
	SCALER     PlanType = "scaler"
	STARTER    PlanType = "starter"
)

var planInfos = map[PlanType]PlanInfo{
	STARTER: {
		maxRowsRead:    1e9,
		maxRowsWritten: 25e6,
		maxDatabases:   3,
		maxLocations:   3,
		maxStorage:     8 * 1024 * 1024 * 1024,
	},
	SCALER: {
		maxRowsRead:    100e9,
		maxRowsWritten: 100e6,
		maxDatabases:   6,
		maxLocations:   6,
		maxStorage:     20 * 1024 * 1024 * 1024,
	},
	ENTERPRISE: {
		maxRowsRead:    0,
		maxRowsWritten: 0,
		maxDatabases:   0,
		maxLocations:   0,
		maxStorage:     0,
	},
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage your account plan and billing",
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountShowCmd)
	accountCmd.AddCommand(accountBookMeetingCmd)
	accountCmd.AddCommand(accountFeedbackCmd)
}

func getPlanInfo(plan PlanType) PlanInfo {
	planInfo := planInfos[plan]

	return planInfo
}
