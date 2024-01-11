package flags

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tursodatabase/turso-cli/internal"
)

var expirationFlag string

func AddExpiration(cmd *cobra.Command) {
	usage := fmt.Sprintf("Token expiration. Possible values are %s (default) or expiration time in days (e.g. %s).", internal.Emph("never"), internal.Emph("7d"))
	cmd.Flags().StringVarP(&expirationFlag, "expiration", "e", "never", usage)
	_ = cmd.RegisterFlagCompletionFunc("expiration", expirationFlagCompletion)
}

func Expiration() (string, error) {
	if err := validateExpiration(expirationFlag); err != nil {
		return "", err
	}
	return expirationFlag, nil
}

func validateExpiration(expiration string) error {
	if len(expiration) == 0 {
		return nil
	}
	if expiration == "none" || expiration == "default" || expiration == "never" {
		return nil
	}
	if !strings.HasSuffix(expiration, "d") {
		return fmt.Errorf("expiration must be either 'never' or in days (e.g. 7d)")
	}
	daysStr := strings.TrimSuffix(expiration, "d")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return err
	}
	if days < 1 {
		return fmt.Errorf("expiration must be at least 1 day")
	}
	return nil
}

func expirationFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if _, err := strconv.Atoi(toComplete); err == nil {
		return []string{toComplete + "d"}, cobra.ShellCompDirectiveDefault
	}

	return []string{"never", "1d", "7d"}, cobra.ShellCompDirectiveDefault
}
