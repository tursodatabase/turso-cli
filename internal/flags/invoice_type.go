package flags

import (
	"fmt"

	"github.com/spf13/cobra"
)

var invoiceType string

func AddInvoiceType(cmd *cobra.Command) {
	cmd.Flags().StringVar(&invoiceType, "type", "issued", "type of the invoice. Possible values: 'all', 'upcoming', 'issued'")
	_ = cmd.RegisterFlagCompletionFunc("type", invoiceTypeFlagCompletion)

}

func InvoiceType() (string, error) {
	if err := validateInvoiceType(invoiceType); err != nil {
		return "", err
	}
	return invoiceType, nil
}

func validateInvoiceType(invoiceType string) error {
	switch invoiceType {
	case "issued", "all", "upcoming", "":
		return nil
	default:
		return fmt.Errorf("type parameter must be either 'all' or 'upcoming' or 'issued'")
	}
}

func invoiceTypeFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"issued", "upcoming", "all"}, cobra.ShellCompDirectiveDefault
}
