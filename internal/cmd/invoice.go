package cmd

import "github.com/spf13/cobra"

func init() {
	orgCmd.AddCommand(invoiceCmd)
}

var invoiceCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Manage Invoices",
}
