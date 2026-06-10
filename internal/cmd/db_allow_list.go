package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var allowListIPsFlag []string
var allowListVpcsFlag []string
var clearAllowListIPsFlag bool
var clearAllowListVpcsFlag bool

func init() {
	dbConfigCmd.AddCommand(dbAllowListCmd)
	dbAllowListCmd.AddCommand(dbShowAllowListCmd)
	dbAllowListCmd.AddCommand(dbSetAllowListCmd)
	dbAllowListCmd.AddCommand(dbClearAllowListCmd)
	dbSetAllowListCmd.Flags().StringSliceVar(&allowListIPsFlag, "ip", nil, "IP address or CIDR block to allow. Can be repeated. Replaces the current list of allowed IPs.")
	dbSetAllowListCmd.Flags().StringSliceVar(&allowListVpcsFlag, "vpc", nil, "AWS VPC endpoint ID (vpce-...) to allow. Can be repeated. Replaces the current list of allowed VPC endpoints.")
	dbClearAllowListCmd.Flags().BoolVar(&clearAllowListIPsFlag, "ips", false, "Clear only the list of allowed IPs")
	dbClearAllowListCmd.Flags().BoolVar(&clearAllowListVpcsFlag, "vpcs", false, "Clear only the list of allowed AWS VPC endpoint IDs")
}

var dbAllowListCmd = &cobra.Command{
	Use:               "allow-list",
	Short:             "Manage the access allow list of a database",
	Long:              "Manage the access allow list of a database. When an allow list is configured, only connections from the listed IPs/CIDRs or AWS VPC endpoints are accepted.",
	ValidArgsFunction: noSpaceArg,
}

var dbShowAllowListCmd = &cobra.Command{
	Use:               "show <database-name>",
	Short:             "Shows the access allow list of a database",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		database, err := getDatabase(client, args[0], true)
		if err != nil {
			return err
		}
		config, err := getDatabaseConfig(client, database.Name)
		if err != nil {
			return err
		}
		fmt.Print(allowListMessage(&config))
		return nil
	},
}

var dbSetAllowListCmd = &cobra.Command{
	Use:   "set <database-name> [--ip <address-or-cidr>]... [--vpc <vpce-id>]...",
	Short: "Sets the access allow list of a database",
	Long: "Sets the access allow list of a database. Each provided flag replaces the corresponding list; " +
		"a list whose flag is not provided is left unchanged.",
	Example: "  turso db config allow-list set my-db --ip 203.0.113.7 --ip 10.0.0.0/8\n" +
		"  turso db config allow-list set my-db --vpc vpce-0fe6c8807461bba49",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		setIPs := cmd.Flags().Changed("ip")
		setVpcs := cmd.Flags().Changed("vpc")
		if !setIPs && !setVpcs {
			return fmt.Errorf("specify at least one of --ip or --vpc. To remove restrictions, use %s", internal.Emph("turso db config allow-list clear"))
		}

		config := turso.DatabaseConfig{}
		if setIPs {
			ips := normalizeAllowListEntries(allowListIPsFlag)
			if err := validateAllowedIPs(ips); err != nil {
				return err
			}
			config.AllowedIPs = &ips
		}
		if setVpcs {
			vpcs := normalizeAllowListEntries(allowListVpcsFlag)
			if err := validateAllowedVpcIDs(vpcs); err != nil {
				return err
			}
			config.AllowedAwsVpcIDs = &vpcs
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		database, err := getDatabase(client, args[0], true)
		if err != nil {
			return err
		}
		if err := client.Databases.UpdateConfig(database.Name, config); err != nil {
			return err
		}
		fmt.Printf("Updated access allow list for database %s\n", internal.Emph(database.Name))
		fmt.Print(allowListMessage(&config))
		return nil
	},
}

var dbClearAllowListCmd = &cobra.Command{
	Use:               "clear <database-name>",
	Short:             "Clears the access allow list of a database, allowing connections from any source",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		clearIPs, clearVpcs := clearAllowListIPsFlag, clearAllowListVpcsFlag
		if !clearIPs && !clearVpcs {
			clearIPs, clearVpcs = true, true
		}

		config := turso.DatabaseConfig{}
		empty := []string{}
		if clearIPs {
			config.AllowedIPs = &empty
		}
		if clearVpcs {
			config.AllowedAwsVpcIDs = &empty
		}

		client, err := authedTursoClient()
		if err != nil {
			return err
		}
		database, err := getDatabase(client, args[0], true)
		if err != nil {
			return err
		}
		if err := client.Databases.UpdateConfig(database.Name, config); err != nil {
			return err
		}

		cleared := []string{}
		if clearIPs {
			cleared = append(cleared, "allowed IPs")
		}
		if clearVpcs {
			cleared = append(cleared, "allowed AWS VPC endpoint IDs")
		}
		fmt.Printf("Cleared %s for database %s\n", strings.Join(cleared, " and "), internal.Emph(database.Name))
		return nil
	},
}

// normalizeAllowListEntries trims whitespace, drops empty entries, and
// removes duplicates while preserving order.
func normalizeAllowListEntries(entries []string) []string {
	seen := make(map[string]bool, len(entries))
	result := []string{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || seen[entry] {
			continue
		}
		seen[entry] = true
		result = append(result, entry)
	}
	return result
}

func validateAllowedIPs(entries []string) error {
	for _, entry := range entries {
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("invalid CIDR block %s. Valid entries are individual IP addresses or CIDR blocks", internal.Emph(entry))
			}
			continue
		}
		if net.ParseIP(entry) == nil {
			return fmt.Errorf("invalid IP address %s. Valid entries are individual IP addresses or CIDR blocks", internal.Emph(entry))
		}
	}
	return nil
}

func validateAllowedVpcIDs(entries []string) error {
	for _, entry := range entries {
		if !strings.HasPrefix(entry, "vpce-") || len(entry) == len("vpce-") {
			return fmt.Errorf("invalid AWS VPC endpoint ID %s: must start with %s", internal.Emph(entry), internal.Emph("vpce-"))
		}
	}
	return nil
}

func allowListMessage(config *turso.DatabaseConfig) string {
	ips := config.AllowedIPList()
	vpcs := config.AllowedVpcIDList()
	if len(ips) == 0 && len(vpcs) == 0 {
		return fmt.Sprintf("Access allow list is %s: connections from any source are accepted\n", internal.Emph("empty"))
	}
	var b strings.Builder
	if len(ips) > 0 {
		b.WriteString("Allowed IPs:\n")
		for _, ip := range ips {
			fmt.Fprintf(&b, "  %s\n", ip)
		}
	}
	if len(vpcs) > 0 {
		b.WriteString("Allowed AWS VPC endpoint IDs:\n")
		for _, vpc := range vpcs {
			fmt.Fprintf(&b, "  %s\n", vpc)
		}
	}
	return b.String()
}
