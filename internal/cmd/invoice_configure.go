package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	invoiceCmd.AddCommand(configureInvoiceInfoCmd)
	addBillingCustomerFlags(configureInvoiceInfoCmd)
}

var configureInvoiceInfoCmd = &cobra.Command{
	Use:               "configure",
	Short:             "Configure billing information added to invoices.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		oldCustomer, err := client.Billing.GetBillingCustomer()
		if err != nil {
			return err
		}

		customer := parseBillingCustomerFlags(oldCustomer)

		err = client.Billing.UpdateBillingCustomer(customer)
		if err != nil {
			return err
		}

		customer, err = client.Billing.GetBillingCustomer()
		if err != nil {
			return err
		}

		fmt.Println("Billing Information:")
		printBillingInfo(customer)

		return nil
	},
}

func parseBillingCustomerFlags(customer turso.BillingCustomer) turso.BillingCustomer {
	if billingCustomerNameFlag != "" {
		customer.Name = billingCustomerNameFlag
	}

	if billingCustomerEmailFlag != "" {
		customer.Email = billingCustomerEmailFlag
	}

	if billingCustomerBillingAddressLine1Flag != "" {
		customer.BillingAddress.Line1 = billingCustomerBillingAddressLine1Flag
	}

	if billingCustomerBillingAddressLine2Flag != "" {
		customer.BillingAddress.Line2 = billingCustomerBillingAddressLine2Flag
	}

	if billingCustomerBillingAddressCityFlag != "" {
		customer.BillingAddress.City = billingCustomerBillingAddressCityFlag
	}

	if billingCustomerBillingAddressStateFlag != "" {
		customer.BillingAddress.State = billingCustomerBillingAddressStateFlag
	}

	if billingCustomerBillingAddressPostalCodeFlag != "" {
		customer.BillingAddress.PostalCode = billingCustomerBillingAddressPostalCodeFlag
	}

	if billingCustomerBillingAddressCountryFlag != "" {
		customer.BillingAddress.Country = billingCustomerBillingAddressCountryFlag
	}

	if billingCustomerTaxIDCountryFlag != "" {
		customer.TaxID.Country = billingCustomerTaxIDCountryFlag
	}

	if billingCustomerTaxIDNumberFlag != "" {
		customer.TaxID.Value = billingCustomerTaxIDNumberFlag
	}
	return customer
}
