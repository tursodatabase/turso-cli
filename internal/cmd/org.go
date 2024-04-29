package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/settings"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var adminFlag bool

func init() {
	rootCmd.AddCommand(orgCmd)
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgDestroyCmd)
	orgCmd.AddCommand(orgSwitchCmd)
	orgCmd.AddCommand(membersCmd)
	orgCmd.AddCommand(invitesCmd)
	membersCmd.AddCommand(membersListCmd)
	membersCmd.AddCommand(membersAddCmd)
	membersCmd.AddCommand(membersRemoveCmd)
	membersCmd.AddCommand(membersInviteCmd)
	invitesCmd.AddCommand(membersInviteCmd)
	invitesCmd.AddCommand(inviteRemoveCmd)
	invitesCmd.AddCommand(inviteListCmd)
	orgCmd.AddCommand(orgBillingCmd)
	membersAddCmd.Flags().BoolVarP(&adminFlag, "admin", "a", false, "Add the user as an admin")
	membersInviteCmd.Flags().BoolVarP(&adminFlag, "admin", "a", false, "Invite the user as an admin")
}

func switchToOrg(client *turso.Client, slug string) error {
	settings, err := settings.ReadSettings()
	if err != nil {
		return err
	}
	orgs, err := client.Organizations.List()
	if err != nil {
		return err
	}

	current := settings.Organization()
	if current == "" {
		for _, o := range orgs {
			if o.Type == "personal" {
				current = o.Slug
				break
			}
		}
	}

	if current == slug {
		fmt.Printf("Organization %s already selected\n", internal.Emph(slug))
		return nil
	}

	prev := fmt.Sprintf("turso org switch %s", current)

	org, err := findOrgWithSlug(orgs, slug)
	if err != nil {
		return err
	}

	if org.Type == "personal" {
		slug = ""
	}

	settings.SetOrganization(slug)

	fmt.Printf("Current organization set to %s.\n", internal.Emph(org.Slug))
	fmt.Printf("All your %s commands will be executed in that organization context.\n", internal.Emph("turso"))
	fmt.Printf("To switch back to your previous organization:\n\n\t%s\n", internal.Emph(prev))
	invalidateDatabasesCache()
	return nil
}

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Manage your organizations",
}

var orgListCmd = &cobra.Command{
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

		client, err := authedTursoClient()
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
	Use:               "create <organization-name>",
	Short:             "Create a new organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		name := args[0]

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		_, err = client.Organizations.Create(name, "", true)
		if err != nil {
			return err
		}

		fmt.Printf("Organizations are only supported in paid plans.\n\n")

		stripeCustomerId, err := client.Billing.CreateStripeCustomer(name)
		if err != nil {
			return fmt.Errorf("failed to create customer: %w", err)
		}
		ok, err := PaymentMethodHelperWithStripeId(client, stripeCustomerId, name)
		if err != nil {
			return fmt.Errorf("failed to add payment method: %w", err)
		}
		if !ok {
			fmt.Println("organization creation aborted")
			return nil
		}
		fmt.Printf("You can manage your payment methods with %s.\n\n", internal.Emph("turso org billing"))
		fmt.Printf("You're creating organization %s on the %s plan.\n", internal.Emph(name), internal.Emph("scaler"))

		ok, err = promptConfirmation("Do you want to continue?")
		if err != nil {
			return fmt.Errorf("could not get prompt confirmed: %w", err)
		}
		if !ok {
			fmt.Println("organization creation aborted")
			return nil
		}
		org, err := client.Organizations.Create(name, stripeCustomerId, false)
		if err != nil {
			return err
		}

		fmt.Printf("\nCreated organization %s.\n", internal.Emph(org.Name))
		switchToOrg(client, org.Name)
		fmt.Println()
		client, err = authedTursoClient()
		if err != nil {
			client.Organizations.Delete(org.Slug)
			return err
		}
		if err = client.Subscriptions.Update("scaler", "", nil); err != nil {
			client.Organizations.Delete(org.Slug)
			return err
		}

		return err
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

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		if settings.Organization() == slug {
			return fmt.Errorf("cannot destroy current organization, please switch to another one first")
		}

		if err = client.Organizations.Delete(slug); err != nil {
			return err
		}
		invalidateDatabasesCache()
		fmt.Printf("Destroyed organization %s.\n", internal.Emph(slug))
		return nil
	},
}

var orgSwitchCmd = &cobra.Command{
	Use:               "switch <organization-slug>",
	Short:             "Switch to an organization as the context for your commands.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: orgSwitchArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		slug := args[0]

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		return switchToOrg(client, slug)
	},
}

func orgSwitchArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return organizationArgs(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
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

func extractOrgNames(orgs []turso.Organization) []string {
	names := make([]string, 0)
	for _, org := range orgs {
		names = append(names, org.Name)
	}
	return names
}

func formatCurrent(org turso.Organization) turso.Organization {
	org.Name = internal.Emph(org.Name)
	org.Slug = fmt.Sprintf("%s (current)", internal.Emph(org.Slug))
	return org
}

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage your organization members",
}

var invitesCmd = &cobra.Command{
	Use:   "invites",
	Short: "Manage your organization invites",
}

var membersListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List members of current organization",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		members, err := client.Organizations.ListMembers()
		if err != nil {
			return err
		}

		data := make([][]string, 0, len(members))
		for _, member := range members {
			data = append(data, []string{member.Name, member.Role})
		}

		printTable([]string{"name", "role"}, data)
		return nil
	},
}

var membersAddCmd = &cobra.Command{
	Use:               "add <username>",
	Short:             "Add a member to current organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		username := args[0]
		if username == "" {
			return fmt.Errorf("username cannot be empty")
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		role := "member"

		if adminFlag {
			role = "admin"
		}

		if err := client.Organizations.AddMember(username, role); err != nil {
			return err
		}

		org := settings.Organization()
		fmt.Printf("User %s added to organization %s.\n", internal.Emph(username), internal.Emph(org))
		return nil
	},
}

var membersInviteCmd = &cobra.Command{
	Use:               "create <email>",
	Aliases:           []string{"invite"},
	Short:             "Invite an email to join the current organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		email := args[0]
		if email == "" {
			return fmt.Errorf("email cannot be empty")
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		role := "member"

		if adminFlag {
			role = "admin"
		}

		if err := client.Organizations.InviteMember(email, role); err != nil {
			return err
		}

		fmt.Printf("Email %s invited.\n", internal.Emph(email))
		return nil
	},
}

var inviteRemoveCmd = &cobra.Command{
	Use:               "remove <email>",
	Short:             "Remove a pending invite to an email to join the current organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		email := args[0]
		if email == "" {
			return fmt.Errorf("email cannot be empty")
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		if err := client.Organizations.DeleteInvite(email); err != nil {
			return err
		}

		fmt.Printf("Pending invite to email %s removed.\n", internal.Emph(email))
		return nil
	},
}

var inviteListCmd = &cobra.Command{
	Use:               "list",
	Short:             "List invites in the current organization",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		invites, err := client.Organizations.ListInvites()
		if err != nil {
			return err
		}

		printInviteTable(invites)

		return nil
	},
}

func printInviteTable(invites []turso.Invite) {
	data := make([][]string, 0, len(invites))
	for _, invite := range invites {
		data = append(data, []string{invite.Email, invite.Role, strconv.FormatBool(invite.Accepted)})
	}

	printTable([]string{"email", "role", "accepted"}, data)
}

var membersRemoveCmd = &cobra.Command{
	Use:               "rm <username>",
	Short:             "Remove a member from the current organization",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		settings, err := settings.ReadSettings()
		if err != nil {
			return err
		}

		username := args[0]
		if username == "" {
			return fmt.Errorf("username cannot be empty")
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		if err := client.Organizations.RemoveMember(username); err != nil {
			return err
		}

		org := settings.Organization()
		fmt.Printf("User %s removed from organization %s.\n", internal.Emph(username), internal.Emph(org))
		return nil
	},
}

var orgBillingCmd = &cobra.Command{
	Use:   "billing",
	Short: "Manange payment methods for the current organization.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		client, err := authedTursoClient()
		if err != nil {
			return err
		}

		return billingPortal(client)
	},
}

func listOrganizations(client *turso.Client, fresh ...bool) ([]turso.Organization, error) {
	skipCache := len(fresh) > 0 && fresh[0]
	if cache := getOrgsCache(); !skipCache && cache != nil {
		return cache, nil
	}
	orgs, err := client.Organizations.List()
	if err != nil {
		return nil, err
	}
	setOrgsCache(orgs)
	return orgs, nil
}
