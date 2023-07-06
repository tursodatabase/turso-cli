package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chiselstrike/turso-cli/internal"
	"github.com/chiselstrike/turso-cli/internal/prompt"
	"github.com/chiselstrike/turso-cli/internal/turso"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/pkg/browser"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planSelectCmd)
	planCmd.AddCommand(planUpgradeCmd)
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
		fmt.Printf("Plan: %s\n", internal.Emph(plan))
		fmt.Println()

		current := getPlan(plan, plans)

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "LIMIT")
		columns = append(columns, "USED %")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		addResourceRowBytes(tbl, "storage", usage.Total.StorageBytesUsed, current.Quotas.Storage)
		addResourceRowMillions(tbl, "rows read", usage.Total.RowsRead, current.Quotas.RowsRead)
		addResourceRowMillions(tbl, "rows written", usage.Total.RowsWritten, current.Quotas.RowsWritten)
		addResourceRowCount(tbl, "databases", usage.Total.Databases, current.Quotas.Databases)
		addResourceRowCount(tbl, "locations", usage.Total.Locations, current.Quotas.Locations)
		tbl.Print()
		fmt.Printf("\nQuota will reset on %s\n", getFirstDayOfNextMonth().Local().Format(time.RFC1123))
		return nil
	},
}

func orgPlanData(client *turso.Client) (sub string, usage turso.OrgUsage, plans []turso.Plan, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		sub, err = client.Subscriptions.Get()
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

		plans, current, hasPaymentMethod, err := getSelectPlanInfo(client)
		if err != nil {
			return fmt.Errorf("failed to get plans: %w", err)
		}

		selected, err := promptPlanSelection(plans, current)
		if err != nil {
			return err
		}

		return changePlan(client, plans, current, hasPaymentMethod, selected)
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

		plans, current, hasPaymentMethod, err := getSelectPlanInfo(client)
		if err != nil {
			return fmt.Errorf("failed to get plans: %w", err)
		}

		if current == "scaler" {
			fmt.Printf("You've already upgraded to %s! 🎉\n", internal.Emph(current))
			fmt.Println()
			fmt.Println("If you need more resources, we're happy to help.")
			fmt.Printf("You can find us at %s or at Discord (%s)\n", internal.Emph("sales@turso.tech"), internal.Emph("https://discord.com/invite/4B5D7hYwub"))
			return nil
		}

		return changePlan(client, plans, current, hasPaymentMethod, "scaler")
	},
}

func changePlan(client *turso.Client, plans []turso.Plan, current string, hasPaymentMethod bool, selected string) error {
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

	change := "downgrading"
	if upgrade {
		change = "upgrading"
	}
	fmt.Printf("You're %s to the %s plan.\n", change, internal.Emph(selected))

	if upgrade && hasPaymentMethod {
		printPricingInfoDisclaimer()
	}

	if ok, _ := promptConfirmation("Do you want to continue?"); !ok {
		fmt.Printf("Plan change cancelled. You're still on %s.\n", internal.Emph(current))
		return nil
	}

	if err := client.Subscriptions.Set(selected); err != nil {
		return err
	}

	fmt.Println()

	change = "downgraded"
	if upgrade {
		change = "upgraded"
	}
	fmt.Printf("You've been %s to plan %s.\n", change, internal.Emph(selected))
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
		time.Sleep(1 * time.Second)
	}
}

func getSelectPlanInfo(client *turso.Client) (plans []turso.Plan, current string, hasPaymentMethod bool, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		plans, err = client.Plans.List()
		return
	})
	g.Go(func() (err error) {
		current, err = client.Subscriptions.Get()
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

func addResourceRowBytes(tbl table.Table, resource string, used, limit uint64) {
	if limit == 0 {
		tbl.AddRow(resource, humanize.Bytes(used), "Unlimited", "")
		return
	}
	tbl.AddRow(resource, humanize.Bytes(used), humanize.Bytes(limit), percentage(float64(used), float64(limit)))
}

func addResourceRowMillions(tbl table.Table, resource string, used, limit uint64) {
	if limit == 0 {
		tbl.AddRow(resource, used, "Unlimited", "")
		return
	}
	tbl.AddRow(resource, toM(used), toM(limit), percentage(float64(used), float64(limit)))
}

func toM(v uint64) string {
	str := fmt.Sprintf("%.1f", float64(v)/1_000_000.0)
	str = strings.TrimSuffix(str, ".0")
	if str == "0" && v != 0 {
		str = "<0.1"
	}
	return str + "M"
}

func addResourceRowCount(tbl table.Table, resource string, used, limit uint64) {
	if limit == 0 {
		tbl.AddRow(resource, used, "Unlimited", "")
		return
	}
	tbl.AddRow(resource, used, limit, percentage(float64(used), float64(limit)))
}

func percentage(used, limit float64) string {
	return fmt.Sprintf("%.0f%%", used/limit*100)
}

func getFirstDayOfNextMonth() time.Time {
	now := time.Now().UTC()
	nextMonth := now.AddDate(0, 1, 0)
	year := nextMonth.Year()
	month := nextMonth.Month()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return firstDay
}
