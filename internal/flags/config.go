package flags

import (
	"github.com/spf13/cobra"
)

var resetConfig bool

func AddResetConfigFlag(cmd *cobra.Command) error {
	cmd.PersistentFlags().BoolVar(&resetConfig, "reset-config", false, "")
	if err := cmd.PersistentFlags().MarkHidden("reset-config"); err != nil {
		return err
	}
	return nil
}

func ResetConfig() bool {
	return resetConfig
}
