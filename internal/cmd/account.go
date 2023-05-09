package cmd

import (
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

type PlanInfo struct {
	maxDatabases uint64
	maxStorage   uint64
	maxLocations uint64
}

type PlanType string

const (
	ENTERPRISE PlanType = "enterprise"
	SCALER     PlanType = "scaler"
	STARTER    PlanType = "starter"
)

var planInfos = map[PlanType]PlanInfo{
	STARTER: {
		maxDatabases: 3,
		maxLocations: 3,
		maxStorage:   8,
	},
	SCALER: {
		maxDatabases: 6,
		maxLocations: 6,
		maxStorage:   20,
	},
	ENTERPRISE: {
		maxDatabases: 0,
		maxLocations: 0,
		maxStorage:   0,
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

type FormattedPlanInfo struct {
	maxDatabases string
	maxStorage   string
	maxLocation  string
}

func getPlanInfo(plan PlanType) FormattedPlanInfo {
	planInfo := planInfos[plan]

	maxStorage := "Unlimited"
	maxDatabases := "Unlimited"
	maxLocation := "Unlimited"

	if plan != ENTERPRISE {
		maxStorage = fmt.Sprint(humanize.IBytes(planInfo.maxStorage * 1024 * 1024 * 1024))
		maxDatabases = fmt.Sprint(planInfo.maxDatabases)
		maxLocation = fmt.Sprint(planInfo.maxLocations)
	}

	return FormattedPlanInfo{
		maxStorage:   maxStorage,
		maxDatabases: maxDatabases,
		maxLocation:  maxLocation,
	}
}
