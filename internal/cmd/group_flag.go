package cmd

import "github.com/spf13/cobra"

var groupFlag string

func addGroupFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&groupFlag, "group", "", "create the database in the specified group")
	cmd.Flags().MarkHidden("group")
}

func addPersistentGroupFlag(cmd *cobra.Command, description string) {
	cmd.PersistentFlags().StringVarP(&groupFlag, "group", "g", "", description)
}
