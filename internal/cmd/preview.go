//go:build preview
// +build preview

package cmd

import (
	"errors"
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
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

		portal, err := client.Organizations.BillingPortal()
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

		plan, err := client.Organizations.Plan()
		if err != nil {
			return err
		}

		usage, err := client.Organizations.Usage()
		if err != nil {
			return err
		}

		fmt.Printf("Current plan: %s\n", internal.Emph(plan.Active))
		if plan.Scheduled != "" {
			fmt.Printf("Starting next month: %s\n", internal.Emph(plan.Scheduled))
		}
		fmt.Println()

		columns := make([]interface{}, 0)
		columns = append(columns, "RESOURCE")
		columns = append(columns, "USED")
		columns = append(columns, "MAX")

		tbl := table.New(columns...)

		columnFmt := color.New(color.FgBlue, color.Bold).SprintfFunc()
		tbl.WithFirstColumnFormatter(columnFmt)

		planInfo := getPlanInfo(PlanType(plan.Active))

		tbl.AddRow("storage", humanize.IBytes(usage.Total.StorageBytesUsed), planInfo.maxStorage)
		tbl.AddRow("rows read", usage.Total.RowsRead, fmt.Sprintf("%d", int(1e9)))
		tbl.AddRow("databases", usage.Total.Databases, planInfo.maxDatabases)
		tbl.AddRow("locations", usage.Total.Locations, planInfo.maxLocation)
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

		plan, err := client.Organizations.SetPlan("starter")
		if err != nil && !errors.Is(err, turso.ErrPaymentRequired) {
			return err
		}
		if errors.Is(err, turso.ErrPaymentRequired) {
			return paymentMethodHelper(client)
		}

		fmt.Printf("Current plan: %s\n", internal.Emph(plan.Active))
		if plan.Scheduled != "" {
			fmt.Printf("Starting next month: %s\n", internal.Emph(plan.Scheduled))
		}

		return nil
	},
}

func paymentMethodHelper(client *turso.Client) error {
	fmt.Println("You need to add a payment method before you can change your plan.")
	ok, _ := promptConfirmation("Want to do it right now?")
	if !ok {
		fmt.Printf("When you're reday, you can use %s to manage your payment methods.\n", internal.Emph("turso org billing"))
		return nil
	}

	portal, err := client.Organizations.BillingPortal()
	if err != nil {
		return err
	}

	if err := browser.OpenURL(portal.URL); err != nil {
		fmt.Println("Access the following URL to manage your payment methods:")
		fmt.Println(portal.URL)
	}

	fmt.Printf("After you've added a payment method, just run %s once more!\n", internal.Emph("turso org billing"))
	return nil
}
