package cmd

import "github.com/spf13/cobra"

var (
	billingCustomerNameFlag                     string
	billingCustomerEmailFlag                    string
	billingCustomerTaxIDCountryFlag             string
	billingCustomerTaxIDNumberFlag              string
	billingCustomerBillingAddressLine1Flag      string
	billingCustomerBillingAddressLine2Flag      string
	billingCustomerBillingAddressCityFlag       string
	billingCustomerBillingAddressStateFlag      string
	billingCustomerBillingAddressPostalCodeFlag string
	billingCustomerBillingAddressCountryFlag    string
)

func addBillingCustomerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&billingCustomerNameFlag, "name", "", "Name of the customer")
	cmd.Flags().StringVar(&billingCustomerEmailFlag, "email", "", "Email of the customer")
	cmd.Flags().StringVar(&billingCustomerTaxIDCountryFlag, "tax-id-country", "", "Country of the tax ID")
	cmd.Flags().StringVar(&billingCustomerTaxIDNumberFlag, "tax-id-number", "", "Number of the tax ID")
	cmd.Flags().StringVar(&billingCustomerBillingAddressLine1Flag, "billing-address-line1", "", "Line 1 of the billing address")
	cmd.Flags().StringVar(&billingCustomerBillingAddressLine2Flag, "billing-address-line2", "", "Line 2 of the billing address")
	cmd.Flags().StringVar(&billingCustomerBillingAddressCityFlag, "billing-address-city", "", "City of the billing address")
	cmd.Flags().StringVar(&billingCustomerBillingAddressStateFlag, "billing-address-state", "", "State of the billing address")
	cmd.Flags().StringVar(&billingCustomerBillingAddressPostalCodeFlag, "billing-address-postal-code", "", "Postal code of the billing address")
	cmd.Flags().StringVar(&billingCustomerBillingAddressCountryFlag, "billing-address-country", "", "Country of the billing address")
}
