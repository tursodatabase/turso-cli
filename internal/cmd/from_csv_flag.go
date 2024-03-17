package cmd

import "github.com/spf13/cobra"

var fromCSVFlag string
var csvTableNameFlag string

func addDbFromCSVFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fromCSVFlag, "from-csv", "", "create the database from a csv file")
}

func addCSVTableNameFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&csvTableNameFlag, "csv-table-name", "", "name of the table in the csv file")
}
