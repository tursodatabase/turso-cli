//go:build preview
// +build preview

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/prompt"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	orgCmd.AddCommand(orgBillingCmd)
	rootCmd.AddCommand(planCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planSelectCmd)
	planCmd.AddCommand(planUpgradeCmd)
}

var orgBillingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Manange payment methods for the current organization.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		return billingPortal(client)
	},
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage your organization plan",
}

var planShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}

		plan, usage, plans, err := orgPlanData(client)
		if err != nil {
			return err
		}

		if client.Org != "" {
			fmt.Printf("Organization: %s\n", internal.Emph(client.Org))
		}
		fmt.Printf("Active plan: %s\n", internal.Emph(plan.Active))
		if plan.Scheduled != "" {
			fmt.Printf("Starting next month: %s\n", internal.Emph(plan.Scheduled))
		}
		fmt.Println()

		current := getPlan(plan.Active, plans)

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "LIMIT")
		columns = append(columns, "PERCENTAGE")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		addResourceRowBytes(tbl, "storage", usage.Total.StorageBytesUsed, current.Quotas.Storage)
		addResourceRowMillions(tbl, "rows read", usage.Total.RowsRead, current.Quotas.RowsRead)
		addResourceRowMillions(tbl, "rows written", usage.Total.RowsWritten, current.Quotas.RowsWritten)
		addResourceRowCount(tbl, "databases", usage.Total.Databases, current.Quotas.Databases)
		addResourceRowCount(tbl, "locations", usage.Total.Locations, current.Quotas.Locations)
		tbl.Print()

		return nil
	},
}

func orgPlanData(client *turso.Client) (plan turso.OrgPlan, usage turso.OrgUsage, plans []turso.Plan, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		plan, err = client.Plans.Get()
		return
	})

	g.Go(func() (err error) {
		usage, err = client.Organizations.Usage()
		return
	})

	g.Go(func() (err error) {
		plans, err = client.Plans.List()
		return
	})
	err = g.Wait()
	return
}

var planSelectCmd = &cobra.Command{
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

		return changePlan(client, plans, plan, hasPaymentMethod, selected)
	},
}

var planUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade your current organization plan",
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

		if current == "scaler" {
			fmt.Printf("You've already upgraded to %s! 🎉\n", internal.Emph(current))
			fmt.Println("If you need a more powerful plan, we're happy to help.")
			fmt.Printf("You can find us at %s or at Discord (%s)\n", internal.Emph("sales@turso.tech"), internal.Emph("https://discord.com/invite/4B5D7hYwub"))
			return nil
		}

		return changePlan(client, plans, plan, hasPaymentMethod, "scaler")
	},
}

func changePlan(client *turso.Client, plans []turso.Plan, plan turso.OrgPlan, hasPaymentMethod bool, selected string) error {
	current := plan.Scheduled
	if plan.Scheduled == "" {
		current = plan.Active
	}

	if selected == current {
		fmt.Println("You're all set! No changes are needed.")
		return nil
	}

	upgrade := isUpgrade(getPlan(current, plans), getPlan(selected, plans))
	if !hasPaymentMethod && upgrade {
		ok, err := paymentMethodHelper(client, selected)
		if err != nil {
			return fmt.Errorf("failed to check payment method: %w", err)
		}
		if !ok {
			return nil
		}
		fmt.Println("Payment method added successfully.")
		fmt.Printf("You can manage your payment methods with %s.\n\n", internal.Emph("turso org billing"))
	}

	if upgrade {
		fmt.Printf("You're upgrading to the %s plan.\n", internal.Emph(selected))
	} else {
		fmt.Printf("You're downgrading your plan to %s.\n", internal.Emph(selected))
		fmt.Printf("Changes will effectively take place at the beginning of next month.\n")
	}

	if upgrade && hasPaymentMethod {
		printPricingInfoDisclaimer()
	}

	if ok, _ := promptConfirmation("Do you want to continue?"); !ok {
		fmt.Printf("Plan change cancelled. You're still on %s.\n", internal.Emph(current))
		return nil
	}

	plan, err := client.Plans.Set(selected)
	if err != nil && !errors.Is(err, turso.ErrPaymentRequired) {
		return err
	}

	fmt.Println()

	if plan.Scheduled != "" {
		fmt.Printf("Starting next month, you will be downgraded to the %s plan.\n", internal.Emph(plan.Scheduled))
		return nil
	}

	fmt.Printf("You've been upgraded to the %s plan 🎉\n", internal.Emph(plan.Active))
	fmt.Printf("To see your new quotas, use %s.\n", internal.Emph("turso plan show"))
	return nil
}

func paymentMethodHelper(client *turso.Client, selected string) (bool, error) {
	fmt.Printf("You need to add a payment method before you can upgrade to the %s plan.\n", internal.Emph(selected))
	printPricingInfoDisclaimer()

	ok, _ := promptConfirmation("Want to add a payment method right now?")
	if !ok {
		fmt.Printf("When you're ready, you can use %s to manage your payment methods.\n", internal.Emph("turso org billing"))
		return false, nil
	}

	fmt.Println()
	if err := billingPortal(client); err != nil {
		return false, err
	}
	fmt.Println()

	spinner := prompt.Spinner("Waiting for you to add a payment method")
	defer spinner.Stop()

	errsInARoW := 0
	for {
		hasPaymentMethod, err := client.Billing.HasPaymentMethod()
		if err != nil {
			errsInARoW += 1
		}
		if errsInARoW > 5 {
			return false, err
		}
		if err == nil {
			errsInARoW = 0
		}
		if hasPaymentMethod {
			return true, nil
		}
		time.Sleep(3 * time.Second)
	}
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

func billingPortal(client *turso.Client) error {
	portal, err := client.Billing.Portal()
	if err != nil {
		return err
	}

	msg := "Opening your browser at:"
	if err := browser.OpenURL(portal.URL); err != nil {
		msg = "Access the following URL to manage your payment methods:"
	}
	fmt.Println(msg)
	fmt.Println(portal.URL)
	return nil
}

func printPricingInfoDisclaimer() {
	fmt.Printf("For information about Turso plans pricing and features, access: %s\n\n", internal.Emph("https://turso.tech/pricing"))
}
