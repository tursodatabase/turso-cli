package flags

import (
	"errors"

	"github.com/spf13/cobra"
)

var csvSeparatorValue string

func AddCSVSeparator(cmd *cobra.Command) {
	usage := "CSV separator character. Must be a single character."
	cmd.Flags().StringVar(&csvSeparatorValue, "csv-separator", ",", usage)
}

func CSVSeparator() (rune, error) {
	if err := validateCSVSeparator(csvSeparatorValue); err != nil {
		return 0, err
	}
	return rune(csvSeparatorValue[0]), nil
}

func validateCSVSeparator(csvSeparatorValue string) error {
	if len(csvSeparatorValue) > 1 {
		return errors.New("csv separator must be a single character")
	}
	if csvSeparatorValue == "" {
		return errors.New("csv separator must not be empty")
	}
	return nil
}
