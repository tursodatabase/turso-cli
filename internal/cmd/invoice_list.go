package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	invoiceCmd.AddCommand(listInvoicesCmd)
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

		invoices, err := client.Invoices.List()
		if err != nil {
			return err
		}

		if len(invoices) == 0 {
			fmt.Println("No invoices found.")
			return nil
		}

		printInvoiceListTable(invoices)
		fmt.Println()
		fmt.Println()
		printInvoiceListPdfs(invoices)
		return nil
	},
}

func printInvoiceListTable(invoices []turso.Invoice) {
	headers, data := invoiceListTable(invoices)
	printTable(headers, data)
}

func printInvoiceListPdfs(invoices []turso.Invoice) {
	headers, data := invoiceListPdfs(invoices)
	printTable(headers, data)
}

func invoiceListTable(invoices []turso.Invoice) ([]string, [][]string) {
	headers := []string{"ID", "Amount Due", "Due Date", "Paid At", "Payment Failed At"}
	data := make([][]string, len(invoices))
	for i, invoice := range invoices {
		data[i] = []string{invoice.Number, invoice.Amount, invoice.DueDate, invoice.PaidAt, invoice.PaymentFailedAt}
	}
	return headers, data
}

func invoiceListPdfs(invoices []turso.Invoice) ([]string, [][]string) {
	headers := []string{"ID", "Link"}
	data := make([][]string, len(invoices))
	for i, invoice := range invoices {
		data[i] = []string{invoice.Number, invoice.InvoicePdf}
	}
	return headers, data
}
