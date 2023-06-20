//go:build preview
// +build preview

package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func init() {
	orgCmd.AddCommand(orgBillingCmd)
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
