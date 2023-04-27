package cmd

import (
	"fmt"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/chiselstrike/iku-turso-cli/internal/settings"
	"github.com/chiselstrike/iku-turso-cli/internal/turso"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(orgsCmd)
	orgsCmd.AddCommand(orgsListCmd)
	orgsCmd.AddCommand(orgCreateCmd)
	orgsCmd.AddCommand(orgDestroyCmd)
	orgsCmd.AddCommand(orgSelectCmd)
}

var orgsCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage your organizations",
}

var orgsListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List your organizations",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		orgs, err := client.Organizations.List()
		if err != nil {
			return err
		}

		current := settings.Organization()

		data := make([][]string, 0, len(orgs))
		for _, org := range orgs {
			if isCurrentOrg(org, current) {
				org = formatCurrent(org)
			}
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
	Use:               "create <name>",
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
	Use:               "destroy <slug>",
	Short:             "Destroy an organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg, // TODO: add orgs autocomplete
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		slug := args[0]

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		if settings.Organization() == slug {
			return fmt.Errorf("cannot destroy current organization, please select another one first")
		}

		if err = client.Organizations.Delete(slug); err != nil {
			return err
		}

		fmt.Printf("Destroyed organization %s.\n", internal.Emph(slug))
		return nil
	},
}

var orgSelectCmd = &cobra.Command{
	Use:               "select <slug>",
	Short:             "Select an organization as the context for your commands.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg, // TODO: add orgs autocomplete
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		slug := args[0]

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		client, err := createTursoClient()
		if err != nil {
			return err
		}

		orgs, err := client.Organizations.List()
		if err != nil {
			return err
		}

		org, err := findOrgWithSlug(orgs, slug)
		if err != nil {
			return err
		}

		if org.Type == "personal" {
			slug = ""
		}

		if err := settings.SetOrganization(slug); err != nil {
			return err
		}

		fmt.Printf("Default organization set to %s.\n", internal.Emph(org.Slug))
		fmt.Printf("All your %s commands will be executed in that organization context.\n", internal.Emph("turso"))
		return nil
	},
}

func findOrgWithSlug(orgs []turso.Organization, slug string) (turso.Organization, error) {
	for _, org := range orgs {
		if org.Slug == slug {
			return org, nil
		}
	}
	return turso.Organization{}, fmt.Errorf("organization with slug %s was not found", internal.Emph(slug))
}

func isCurrentOrg(org turso.Organization, currentSlug string) bool {
	if org.Type == "personal" {
		return currentSlug == ""
	}
	return org.Slug == currentSlug
}

func formatCurrent(org turso.Organization) turso.Organization {
	org.Name = internal.Emph(org.Name)
	org.Slug = fmt.Sprintf("%s (current)", internal.Emph(org.Slug))
	return org
}
