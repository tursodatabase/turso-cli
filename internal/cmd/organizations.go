package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(orgsCmd)
	orgsCmd.AddCommand(orgsListCmd)
	orgsCmd.AddCommand(orgCreateCmd)
	orgsCmd.AddCommand(orgDestroyCmd)
}

var orgsCmd = &cobra.Command{
	Use:   "organizations",
	Short: "Manage your organizations",
}

var orgsListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List your organizations",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClient()
		if err != nil {
			return err
		}

		orgs, err := client.Organizations.List()
		if err != nil {
			return err
		}

		data := make([][]string, 0, len(orgs))
		for _, org := range orgs {
			data = append(data, []string{org.Name, org.Slug})
		}

		if len(data) == 0 {
			fmt.Println("You don't have any organizations.")
			return nil
		}

		printTable([]string{"name", "slug"}, data)
		return nil
	},
}

var orgCreateCmd = &cobra.Command{
	Use:               "create",
	Short:             "Create a new organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name := args[0]

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		org, err := client.Organizations.Create(name)
		if err != nil {
			return err
		}

		fmt.Printf("Created organization %s.\n", internal.Emph(org.Name))
		return nil
	},
}

var orgDestroyCmd = &cobra.Command{
	Use:               "destroy",
	Short:             "Destroy an organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg, // TODO: add orgs autocomplete
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name := args[0]

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		if err = client.Organizations.Delete(name); err != nil {
			return err
		}

		fmt.Printf("Destroyed organization %s.\n", internal.Emph(name))
		return nil
	},
}
