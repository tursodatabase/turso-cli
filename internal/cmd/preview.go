//go:build preview
// +build preview

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	orgCmd.AddCommand(orgBillingCmd)
	orgCmd.AddCommand(orgPlanCmd)
	orgPlanCmd.AddCommand(orgPlanShowCmd)
	orgPlanCmd.AddCommand(orgPlanSelectCmd)
}

var orgBillingCmd = &cobra.Command{
	Use:   "billing",
	Short: "manange payment methods of the current organization.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		portal, err := client.Billing.Portal()
		if err != nil {
			return err
		}

		if err := browser.OpenURL(portal.URL); err != nil {
			fmt.Println("Access the following URL to manage your payment methods:")
			fmt.Println(portal.URL)
		}

		return nil
	},
}

var orgPlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage your organization plan",
}

var orgPlanShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		plan, err := client.Plans.Get()
		if err != nil {
			return err
		}

		usage, err := client.Organizations.Usage()
		if err != nil {
			return err
		}

		fmt.Printf("Active plan: %s\n", internal.Emph(plan.Active))
		if plan.Scheduled != "" {
			fmt.Printf("Starting next month: %s\n", internal.Emph(plan.Scheduled))
		}
		fmt.Println()

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "LIMIT")
		columns = append(columns, "PERCENTAGE")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		planInfo := getPlanInfo(PlanType(plan.Active))

		maxStorage, err := humanize.ParseBytes(planInfo.maxStorage)
		if err != nil {
			return err
		}
		maxDatabases, err := strconv.ParseUint(planInfo.maxDatabases, 10, 64)
		if err != nil {
			return err
		}
		maxLocations, err := strconv.ParseUint(planInfo.maxLocation, 10, 64)
		if err != nil {
			return err
		}
		addResourceRowBytes(tbl, "storage", usage.Total.StorageBytesUsed, maxStorage)
		addResourceRowCount(tbl, "rows read", usage.Total.RowsRead, uint64(1e9))
		addResourceRowCount(tbl, "rows written", usage.Total.RowsWritten, uint64(25*1e6))
		addResourceRowCount(tbl, "databases", usage.Total.Databases, maxDatabases)
		addResourceRowCount(tbl, "locations", usage.Total.Locations, maxLocations)
		tbl.Print()

		return nil
	},
}

var orgPlanSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Change your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		plans, plan, hasPaymentMethod, err := getSelectPlanInfo(client)
		if err != nil {
			return fmt.Errorf("failed to get plans: %w", err)
		}

		current := plan.Scheduled
		if plan.Scheduled == "" {
			current = plan.Active
		}

		selected, err := promptPlanSelection(plans, current)
		if err != nil {
			return err
		}

		if selected == current {
			fmt.Println("You're all set! No changes are needed.")
			return nil
		}

		upgrade := isUpgrade(getPlan(current, plans), getPlan(selected, plans))
		if !hasPaymentMethod && upgrade {
			return paymentMethodHelper(client)
		}

		if upgrade {
			fmt.Printf("You're upgrading your plan to paid plan %s.\n", internal.Emph(selected))
			fmt.Printf("For information about resouce quotas and pricing, access: %s\n", internal.Emph("https://turso.tech/pricing"))
		} else {
			fmt.Printf("You're downgrading your plan to %s.\n", internal.Emph(selected))
			fmt.Printf("Changes will effectively take place at the beginning of next month.\n")
		}

		if ok, _ := promptConfirmation("Do you want to continue?"); !ok {
			fmt.Printf("Plan change cancelled. You're still on %s.\n", internal.Emph(current))
			return nil
		}

		plan, err = client.Plans.Set(selected)
		if err != nil && !errors.Is(err, turso.ErrPaymentRequired) {
			return err
		}
		if errors.Is(err, turso.ErrPaymentRequired) {
			return paymentMethodHelper(client)
		}

		fmt.Printf("Active plan: %s\n", internal.Emph(plan.Active))
		if plan.Scheduled != "" {
			fmt.Printf("Starting next month: %s\n", internal.Emph(plan.Scheduled))
		}

		return nil
	},
}

func paymentMethodHelper(client *turso.Client) error {
	fmt.Println("You need to add a payment method before you can upgrade your plan.")
	ok, _ := promptConfirmation("Want to do it right now?")
	if !ok {
		fmt.Printf("When you're ready, you can use %s to add a payment method.\n", internal.Emph("turso org billing"))
		return nil
	}

	portal, err := client.Billing.Portal()
	if err != nil {
		return err
	}

	if err := browser.OpenURL(portal.URL); err != nil {
		fmt.Println("Access the following URL to manage your payment methods:")
		fmt.Println(portal.URL)
	}

	fmt.Printf("After you've added a payment method, just run %s once more!\n", internal.Emph("turso org plan select"))
	return nil
}

func getSelectPlanInfo(client *turso.Client) (plans []turso.Plan, current turso.OrgPlan, hasPaymentMethod bool, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		plans, err = client.Plans.List()
		return
	})
	g.Go(func() (err error) {
		current, err = client.Plans.Get()
		return
	})
	g.Go(func() (err error) {
		hasPaymentMethod, err = client.Billing.HasPaymentMethod()
		return
	})
	err = g.Wait()
	return
}

func promptPlanSelection(plans []turso.Plan, current string) (string, error) {
	planNames := make([]string, 0, len(plans))
	cur := 0
	for _, plan := range plans {
		if plan.Name == current {
			cur = len(planNames)
			planNames = append(planNames, fmt.Sprintf("%s (current)", internal.Emph(plan.Name)))
			continue
		}
		planNames = append(planNames, plan.Name)
	}

	prompt := promptui.Select{
		CursorPos:    cur,
		HideHelp:     true,
		Label:        "Select a plan",
		Items:        planNames,
		HideSelected: true,
	}

	_, result, err := prompt.Run()
	if strings.HasSuffix(result, "(current)") {
		result = current
	}
	return result, err
}

func formatPrice(price string) string {
	return "$" + price
}

func isUpgrade(current, selected turso.Plan) bool {
	cp, _ := strconv.Atoi(current.Price)
	sp, _ := strconv.Atoi(selected.Price)
	return sp > cp
}

func getPlan(name string, plans []turso.Plan) turso.Plan {
	for _, plan := range plans {
		if plan.Name == name {
			return plan
		}
	}
	return turso.Plan{}
}
