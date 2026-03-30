package cmd

import "github.com/spf13/cobra"

var tursoDBFlag bool

func addTursoDBFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&tursoDBFlag, "tursodb", false, "Create the database using TursoDB (MVCC).")
}
