package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

type expirationFlag string

var expFlagValues = []string{"default", "none"}

func (e *expirationFlag) String() string {
	return string(*e)
}

func (e *expirationFlag) Set(v string) error {
	if v == "" {
		*e = "none"
		return nil
	}
	if slices.Contains(expFlagValues, v) {
		*e = expirationFlag(v)
		return nil
	}
	return fmt.Errorf("must be one of: %s", strings.Join(expFlagValues, ", "))
}

func (e *expirationFlag) Type() string {
	return "expiration"
}

func expirationFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"default\tuses the default expiration chosen by the auth server. Ideal for local develpment.",
		"none\tdisables token expiration. Ideal for generating tokens for services.",
	}, cobra.ShellCompDirectiveDefault
}
