package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(planCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planSelectCmd)
	planCmd.AddCommand(planUpgradeCmd)
	planCmd.AddCommand(overagesCommand)
	overagesCommand.AddCommand(planEnableOverages)
	overagesCommand.AddCommand(planDisableOverages)
	flags.AddOverages(planSelectCmd)
	flags.AddTimeline(planSelectCmd)
}

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Manage your organization plan",
}

var overagesCommand = &cobra.Command{
	Use:   "overages",
	Short: "Manage your current organization overages",
}

func getCurrentOrg(client *turso.Client, organizationName string) (turso.Organization, error) {
	orgs, err := client.Organizations.List()
	if err != nil {
		return turso.Organization{}, err
	}
	for _, org := range orgs {
		if org.Slug == organizationName {
			return org, nil
		}
		if organizationName == "" && org.Type == "personal" {
			return org, nil
		}
	}
	return turso.Organization{}, fmt.Errorf("could not find organization %s", organizationName)
}

var planShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		settings, err := settings.ReadSettings()
		if err != nil {
			return fmt.Errorf("could not retrieve local config: %w", err)
		}

		subscription, orgUsage, plans, err := orgPlanData(client)
		if err != nil {
			return err
		}

		plan := subscription.Plan

		var organizationName string
		if organizationName = client.Org; organizationName == "" {
			organizationName = settings.GetUsername()
		}

		currentOrg, err := getCurrentOrg(client, organizationName)
		if err != nil {
			return err
		}

		fmt.Printf("Organization: %s\n", internal.Emph(currentOrg.Name))
		if currentOrg.Overages {
			plan, _ = strings.CutSuffix(plan, "_overages")
		}

		fmt.Printf("Plan: %s\n", internal.Emph(plan))
		fmt.Print(overagesMessage(currentOrg.Overages))
		fmt.Println()

		current := getPlan(plan, plans)
		tbl := planUsageTable(orgUsage, current, currentOrg)
		tbl.Print()
		fmt.Printf("\nQuota will reset on %s\n", getFirstDayOfNextMonth().Local().Format(time.RFC1123))
		return nil
	},
}

func overagesMessage(overages bool) string {
	status := "disabled"
	if overages {
		status = "enabled"
	}
	return fmt.Sprintf("Overages %s\n", internal.Emph(status))
}

func planUsageTable(orgUsage turso.OrgUsage, current turso.Plan, currentOrg turso.Organization) table.Table {
	columns := make([]interface{}, 0)
	columns = append(columns, "RESOURCE")
	columns = append(columns, "USED")

	columns = append(columns, "LIMIT")
	columns = append(columns, "LIMIT %")
	if currentOrg.Overages {
		columns = append(columns, "OVERAGE")
	}

	tbl := table.New(columns...)

	columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
	tbl.WithFirstColumnFormatter(columnFmt)

	addResourceRowBytes(tbl, "storage", orgUsage.Usage.StorageBytesUsed, current.Quotas.Storage, currentOrg.Overages)
	addResourceRowMillions(tbl, "rows read", orgUsage.Usage.RowsRead, current.Quotas.RowsRead, currentOrg.Overages)
	addResourceRowMillions(tbl, "rows written", orgUsage.Usage.RowsWritten, current.Quotas.RowsWritten, currentOrg.Overages)
	addResourceRowBytes(tbl, "embedded syncs", orgUsage.Usage.BytesSynced, current.Quotas.BytesSynced, currentOrg.Overages)
	addResourceRowCount(tbl, "databases", orgUsage.Usage.Databases, current.Quotas.Databases)
	addResourceRowCount(tbl, "locations", orgUsage.Usage.Locations, current.Quotas.Locations)
	addResourceRowCount(tbl, "groups", orgUsage.Usage.Groups, current.Quotas.Groups)
	return tbl
}

func orgPlanData(client *turso.Client) (sub turso.Subscription, usage turso.OrgUsage, plans []turso.Plan, err error) {
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
	Use:               "select",
	Short:             "Change your current organization plan",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: planNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		return BillingPortal()
	},
}

func planNameArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	client, err := authedTursoClient()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	plans, err := getPlans(client)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, 0, len(plans))
	for _, plan := range plans {
		names = append(names, plan.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

var planUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return BillingPortal()
	},
}

var planEnableOverages = &cobra.Command{
	Use:   "enable",
	Short: "Enable overages for your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return BillingPortal()
	},
}

var planDisableOverages = &cobra.Command{
	Use:   "disable",
	Short: "Disable overages for your current organization plan",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return BillingPortal()
	},
}

func PaymentMethodHelper(client *turso.Client, selected string) (bool, error) {
	fmt.Printf("You need to add a payment method before you can upgrade to the %s plan.\n", internal.Emph(selected))
	printPricingInfoDisclaimer()

	ok, _ := promptConfirmation("Want to add a payment method right now?")
	if !ok {
		fmt.Printf("When you're ready, you can use %s to manage your payment methods.\n", internal.Emph("turso org billing"))
		return false, nil
	}

	fmt.Println()
	if err := billingPortal("client"); err != nil {
		return false, err
	}
	fmt.Println()

	spinner := prompt.Spinner("Waiting for you to add a payment method")
	defer spinner.Stop()

	return checkPaymentMethod(client, "")
}

func hasPaymentMethodCheck(client *turso.Client, stripeId string) (bool, error) {
	if stripeId != "" {
		return client.Billing.HasPaymentMethodWithStripeId(stripeId)
	}
	return client.Billing.HasPaymentMethod()
}

func checkPaymentMethod(client *turso.Client, stripeId string) (bool, error) {
	errsInARoW := 0
	var hasPaymentMethod bool
	var err error
	for {
		hasPaymentMethod, err = hasPaymentMethodCheck(client, stripeId)
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

func PaymentMethodHelperOverages(client *turso.Client) (bool, error) {
	fmt.Print("You need to add a payment method before you can enable overages.\n")
	printPricingInfoDisclaimer()

	ok, _ := promptConfirmation("Want to add a payment method right now?")
	if !ok {
		fmt.Printf("When you're ready, you can use %s to manage your payment methods.\n", internal.Emph("turso org billing"))
		return false, nil
	}

	fmt.Println()
	if err := billingPortal("client"); err != nil {
		return false, err
	}
	fmt.Println()

	spinner := prompt.Spinner("Waiting for you to add a payment method")
	defer spinner.Stop()

	return checkPaymentMethod(client, "")
}

func PaymentMethodHelperWithStripeId(client *turso.Client, stripeId, orgName string) (bool, error) {
	fmt.Printf("You need to add a payment method before you can create organization %s on the %s plan.\n", internal.Emph(orgName), internal.Emph("scaler"))
	printPricingInfoDisclaimer()

	ok, _ := promptConfirmation("Want to add a payment method right now?")
	if !ok {
		fmt.Printf("When you're ready, you can use %s to manage your payment methods.\n", internal.Emph("turso org billing"))
		return false, nil
	}

	fmt.Println()
	if err := BillingPortalForStripeId(client, stripeId); err != nil {
		return false, err
	}
	fmt.Println()

	spinner := prompt.Spinner("Waiting for you to add a payment method")
	defer spinner.Stop()

	return checkPaymentMethod(client, stripeId)
}

func GetSelectPlanInfo(client *turso.Client) (plans []turso.Plan, current turso.Subscription, hasPaymentMethod bool, err error) {
	g := errgroup.Group{}
	g.Go(func() (err error) {
		plans, err = getPlans(client)
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

func getPlans(client *turso.Client) ([]turso.Plan, error) {
	if cached := getPlansCache(); cached != nil {
		return cached, nil
	}
	plans, err := client.Plans.List()
	if err != nil {
		return nil, err
	}
	setPlansCache(plans)
	return plans, nil
}

func getPlan(name string, plans []turso.Plan) turso.Plan {
	for _, plan := range plans {
		if plan.Name == name {
			return plan
		}
	}
	return turso.Plan{}
}

func billingPortal(currentOrg string) error {
	url := "https://app.turso.tech/" + currentOrg + "/settings/billing"
	msg := "Opening your browser at:"
	if err := browser.OpenURL(url); err != nil {
		msg = "Access the following URL to manage your payment methods:"
	}
	fmt.Println(msg)
	fmt.Println(url)
	return nil
}

func BillingPortalForStripeId(client *turso.Client, stripeCustomerId string) error {
	portal, err := client.Billing.PortalForStripeId(stripeCustomerId)
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

func addResourceRowBytes(tbl table.Table, resource string, used, limit uint64, overages bool) {
	if limit == 0 {
		tbl.AddRow(resource, humanize.Bytes(used), "Unlimited", "")
		return
	}
	exceededValue := uint64(max(int(used)-int(limit), 0))
	if overages && exceededValue > 0 {
		tbl.AddRow(resource, humanize.Bytes(used), humanize.Bytes(limit), percentage(float64(used), float64(limit)), humanize.Bytes(exceededValue))
		return
	}
	tbl.AddRow(resource, humanize.Bytes(used), humanize.Bytes(limit), percentage(float64(used), float64(limit)))
}

func addResourceRowMillions(tbl table.Table, resource string, used, limit uint64, overages bool) {
	if limit == 0 {
		tbl.AddRow(resource, used, "Unlimited", "")
		return
	}
	exceededValue := uint64(max(int(used)-int(limit), 0))
	if overages && exceededValue > 0 {
		tbl.AddRow(resource, toM(used), toM(limit), percentage(float64(used), float64(limit)), toM(exceededValue))
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
