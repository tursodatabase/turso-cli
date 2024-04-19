package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal/prompt"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

func init() {
	invoiceCmd.AddCommand(configureInvoiceInfoCmd)
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

		customer, err := promptConfigureBillingCustomer(oldCustomer)
		if err != nil {
			return err
		}

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

func promptConfigureBillingCustomer(customer turso.BillingCustomer) (turso.BillingCustomer, error) {
	newName, err := prompt.TextInput("Name", customer.Name, "")
	if err != nil {
		return customer, err
	}
	customer.Name = newName

	newEmail, err := prompt.TextInput("Email", customer.Email, "")
	if err != nil {
		return customer, err
	}
	customer.Email = newEmail

	newBillingAddressLine1, err := prompt.TextInput("Billing Address Line 1", customer.BillingAddress.Line1, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.Line1 = newBillingAddressLine1

	newBillingAddressLine2, err := prompt.TextInput("Billing Address Line 2", customer.BillingAddress.Line2, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.Line2 = newBillingAddressLine2

	newBillingAddressCity, err := prompt.TextInput("Billing Address City", customer.BillingAddress.City, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.City = newBillingAddressCity

	newBillingAddressState, err := prompt.TextInput("Billing Address State", customer.BillingAddress.State, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.State = newBillingAddressState

	newBillingAddressPostalCode, err := prompt.TextInput("Billing Address Postal Code", customer.BillingAddress.PostalCode, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.PostalCode = newBillingAddressPostalCode

	newBillingAddressCountry, err := prompt.TextInput("Billing Address Country", customer.BillingAddress.Country, "")
	if err != nil {
		return customer, err
	}
	customer.BillingAddress.Country = newBillingAddressCountry

	newTaxIDCountry, err := prompt.TextInput("Tax ID Country", customer.TaxID.Country, "")
	if err != nil {
		return customer, err
	}
	customer.TaxID.Country = newTaxIDCountry

	newTaxIDNumber, err := prompt.TextInput("Tax ID Number", customer.TaxID.Value, "")
	if err != nil {
		return customer, err
	}
	customer.TaxID.Value = newTaxIDNumber

	return customer, nil
}
