package flags

import (
	"fmt"

	"github.com/spf13/cobra"
)

var csvSeparator rune
var csvSeparatorValue string

func AddCSVSeparator(cmd *cobra.Command) {
	usage := "CSV separator character. Must be a single character. Defaults to ','"
	cmd.Flags().StringVar(&csvSeparatorValue, "csv-separator", ",", usage)
}

func CSVSeparator() (rune, error) {
	if err := validateCSVSeparator(csvSeparatorValue); err != nil {
		return 0, err
	}
    csvSeparator = rune(csvSeparatorValue[0])
	return csvSeparator, nil
}

func validateCSVSeparator(csvSeparatorValue string) error {
	if len(csvSeparatorValue) > 1 {
		return fmt.Errorf("csv separator must be a single character")
	}
	if csvSeparatorValue == "" {
		return fmt.Errorf("csv separator must not be empty")
	}
	return nil
}
