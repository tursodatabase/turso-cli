package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

func init() {
	dbCmd.AddCommand(dbInspectCmd)
}

var dbInspectCmd = &cobra.Command{
	Use:               "inspect {database_name}",
	Short:             "Inspect database.",
	Example:           "turso db inspect name-of-my-amazing-db",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: dbNameArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("please specify a database name")
		}
		cmd.SilenceUsage = true
		dbUrl, err := getDatabaseURL(name)
		if err != nil {
			return err
		}
		return inspect(dbUrl)
	},
}

func inspect(url string) error {
	stmt := "select * from dbstat"
	resp, err := doQuery(url, stmt)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: %s", string(body))
	}

	var results []QueryResult
	if err := json.Unmarshal(body, &results); err != nil {
		return err
	}
	errs := []string{}
	for _, result := range results {
		if result.Error != nil {
			errs = append(errs, result.Error.Message)
		}
		if result.Results != nil {
			sizes := map[string]float64{}
			for _, row := range result.Results.Rows {
				name := row[0].(string)
				size := row[9].(float64)
				sizes[name] += size
			}
			columns := make([]interface{}, 0)
			columns = append(columns, "TABLE/INDEX")
			columns = append(columns, "SIZE (KB)")
			tbl := table.New(columns...)
			for name, size := range sizes {
				tbl.AddRow(name, size/1024.0)
			}
			tbl.Print()
		}
	}
	if len(errs) > 0 {
		return &SqlError{(strings.Join(errs, "; "))}
	}

	return nil
}
