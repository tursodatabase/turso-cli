package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/flags"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	invoiceCmd.AddCommand(listInvoicesCmd)
	flags.AddInvoiceType(listInvoicesCmd)
}

var listInvoicesCmd = &cobra.Command{
	Use:               "list",
	Short:             "List invoices.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		invoiceType, err := flags.InvoiceType()
		if err != nil {
			return err
		}

		invoices, err := client.Invoices.List(invoiceType)
		if err != nil {
			return err
		}

		if len(invoices) == 0 {
			fmt.Println("No invoices found.")
			return nil
		}

		printInvoiceTable(invoices)
		fmt.Println()
		fmt.Println()
		printInvoiceLinks(invoices)
		return nil
	},
}

func printInvoiceTable(invoices []turso.Invoice) {
	headers, data := invoiceTable(invoices)
	printTable(headers, data)
}

func printInvoiceLinks(invoices []turso.Invoice) {
	headers, data := invoiceLinks(invoices)
	printTable(headers, data)
}

func invoiceTable(invoices []turso.Invoice) ([]string, [][]string) {
	headers := []string{"ID", "Amount Due", "Status", "Due Date", "Paid At", "Payment Failed At"}
	data := make([][]string, len(invoices))
	for i, invoice := range invoices {
		data[i] = []string{invoice.Number, invoice.Amount, invoice.Status, invoice.DueDate, invoice.PaidAt, invoice.PaymentFailedAt}
	}
	return headers, data
}

func invoiceLinks(invoices []turso.Invoice) ([]string, [][]string) {
	headers := []string{"ID", "Link"}
	data := make([][]string, len(invoices))
	for i, invoice := range invoices {
		invoiceLink := invoice.InvoicePdf
		if invoice.InvoicePdf == "" {
			invoiceLink = invoice.HostedInvoiceUrl
		}
		data[i] = []string{invoice.Number, invoiceLink}
	}
	return headers, data
}
