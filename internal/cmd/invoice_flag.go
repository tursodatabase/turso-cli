package cmd

import "github.com/spf13/cobra"

var invoiceType string

func AddInvoiceType(cmd *cobra.Command) {
	cmd.Flags().StringVar(&invoiceType, "type", "", "type of the invoice. Possible values: 'all', 'upcoming', 'issued'")
}
