package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/spf13/cobra"
)

type expirationFlag string

func (e *expirationFlag) String() string {
	return string(*e)
}

func (e *expirationFlag) Set(v string) error {
	if v == "" || v == "never" {
		*e = "never"
		return nil
	}

	if strings.HasSuffix(v, "d") {
		checkIfNumber := strings.TrimSuffix(v, "d")
		if _, err := strconv.Atoi(checkIfNumber); err == nil {
			*e = expirationFlag(v)
			return nil
		}
	}
	return fmt.Errorf("must be %s or expiration time in days (e.g. %s)", internal.Emph("never"), internal.Emph("7d"))
}

func (e *expirationFlag) Type() string {
	return "expiration"
}

func expirationFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"never\tdisables token expiration. Ideal for generating tokens for services.",
	}, cobra.ShellCompDirectiveDefault
}
