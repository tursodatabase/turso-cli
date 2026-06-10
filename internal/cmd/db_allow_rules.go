package cmd

import (
	"fmt"
	"net"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

var allowRulesIPsFlag []string
var allowRulesVpcsFlag []string
var clearAllowRulesIPsFlag bool
var clearAllowRulesVpcsFlag bool

func init() {
	dbConfigCmd.AddCommand(dbAllowRulesCmd)
	dbAllowRulesCmd.AddCommand(dbShowAllowRulesCmd)
	dbAllowRulesCmd.AddCommand(dbSetAllowRulesCmd)
	dbAllowRulesCmd.AddCommand(dbClearAllowRulesCmd)
	dbSetAllowRulesCmd.Flags().StringSliceVar(&allowRulesIPsFlag, "ip", nil, "IP address or CIDR block to allow. Can be repeated. Replaces the current list of allowed IPs.")
	dbSetAllowRulesCmd.Flags().StringSliceVar(&allowRulesVpcsFlag, "aws-vpc", nil, "AWS VPC endpoint ID (vpce-...) to allow. Can be repeated. Replaces the current list of allowed VPC endpoints.")
	dbClearAllowRulesCmd.Flags().BoolVar(&clearAllowRulesIPsFlag, "ips", false, "Clear only the list of allowed IPs")
	dbClearAllowRulesCmd.Flags().BoolVar(&clearAllowRulesVpcsFlag, "aws-vpcs", false, "Clear only the list of allowed AWS VPC endpoint IDs")
}

var dbAllowRulesCmd = &cobra.Command{
	Use:   "allow-rules",
	Short: "Manage the access allow rules of a database",
	Long: "Manage the access allow rules of a database. A connection must satisfy every configured rule list: " +
		"if allowed IPs are set, the client IP must be on the list; if allowed AWS VPC endpoints are set, " +
		"the connection must arrive through one of them.",
	ValidArgsFunction: noSpaceArg,
}

var dbShowAllowRulesCmd = &cobra.Command{
	Use:               "show <database-name>",
	Short:             "Shows the access allow rules of a database",
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
		fmt.Print(allowRulesMessage(&config))
		return nil
	},
}

var dbSetAllowRulesCmd = &cobra.Command{
	Use:   "set <database-name> [--ip <address-or-cidr>]... [--aws-vpc <vpce-id>]...",
	Short: "Sets the access allow rules of a database",
	Long: "Sets the access allow rules of a database. Each provided flag replaces the corresponding list; " +
		"a list whose flag is not provided is left unchanged. When both lists are set, connections must " +
		"satisfy both: an allowed IP arriving through an allowed AWS VPC endpoint.",
	Example: "  turso db config allow-rules set my-db --ip 203.0.113.7 --ip 10.0.0.0/8\n" +
		"  turso db config allow-rules set my-db --aws-vpc vpce-0fe6c8807461bba49",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		setIPs := cmd.Flags().Changed("ip")
		setVpcs := cmd.Flags().Changed("aws-vpc")
		if !setIPs && !setVpcs {
			return fmt.Errorf("specify at least one of --ip or --aws-vpc. To remove restrictions, use %s", internal.Emph("turso db config allow-rules clear"))
		}

		config := turso.DatabaseConfig{}
		if setIPs {
			ips := normalizeAllowRuleEntries(allowRulesIPsFlag)
			if err := validateAllowedIPs(ips); err != nil {
				return err
			}
			config.AllowedIPs = &ips
		}
		if setVpcs {
			vpcs := normalizeAllowRuleEntries(allowRulesVpcsFlag)
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
		fmt.Printf("Updated access allow rules for database %s\n", internal.Emph(database.Name))
		fmt.Print(allowRulesMessage(&config))
		return nil
	},
}

var dbClearAllowRulesCmd = &cobra.Command{
	Use:               "clear <database-name>",
	Short:             "Clears the access allow rules of a database, allowing connections from any source",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		clearIPs, clearVpcs := clearAllowRulesIPsFlag, clearAllowRulesVpcsFlag
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

// normalizeAllowRuleEntries trims whitespace, drops empty entries, and
// removes duplicates while preserving order.
func normalizeAllowRuleEntries(entries []string) []string {
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

func allowRulesMessage(config *turso.DatabaseConfig) string {
	ips := config.AllowedIPList()
	vpcs := config.AllowedVpcIDList()
	if len(ips) == 0 && len(vpcs) == 0 {
		return fmt.Sprintf("Access allow rules are %s: connections from any source are accepted\n", internal.Emph("empty"))
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
