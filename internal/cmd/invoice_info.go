package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	invoiceCmd.AddCommand(showInvoiceInfoCmd)
}

var showInvoiceInfoCmd = &cobra.Command{
	Use:               "info",
	Short:             "Show billing information added to invoices.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		customer, err := client.Billing.GetBillingCustomer()
		if err != nil {
			return err
		}

		fmt.Println("Billing Information:")
		printBillingInfo(customer)

		return nil
	},
}

func printBillingInfo(customer turso.BillingCustomer) {
	fmt.Println("Name:", customer.Name)
	fmt.Println("Email:", customer.Email)
	fmt.Println()
	fmt.Println("Billing Address:")
	fmt.Println("Line 1:", customer.BillingAddress.Line1)
	fmt.Println("Line 2:", customer.BillingAddress.Line2)
	fmt.Println("City:", customer.BillingAddress.City)
	fmt.Println("State:", customer.BillingAddress.State)
	fmt.Println("Zip:", customer.BillingAddress.PostalCode)
	fmt.Println("Country:", customer.BillingAddress.Country)
	fmt.Println()
	fmt.Println("Tax Information:")
	fmt.Println("VAT:", customer.TaxID.Value)
	fmt.Println("Tax Country:", customer.TaxID.Country)
}
